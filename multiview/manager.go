package multiview

import (
	"DSM-project/memory"
	"DSM-project/network"
	"DSM-project/treadmarks"
	"fmt"
	"log"
	"sync"
)

//Each minipage consists of an offset and a length.
type minipage struct {
	offset, length int
}

//this is the actual manager.
type Manager struct {
	treadmarks.LockManager
	treadmarks.BarrierManager
	copyLock *sync.RWMutex
	tr       network.ITransciever
	cl       *network.Client      //The transciever that we are sending messages over.
	vm       memory.VirtualMemory //The virtual memory object we are working on in the system.
	mpt      map[int]minipage     //Minipagetable
	log      map[int]int          //A map, where each entrance points to
	// the first vpage of this allocation. Used for freeing.
	copies map[int][]byte        //A map of who has copies of what vpage
	locks  map[int]*sync.RWMutex //A map of locks belonging to each vpage.
	*sync.Mutex
}

// Returns the pointer to a manager object.
func NewManager(vm memory.VirtualMemory) *Manager {
	m := Manager{
		copyLock: new(sync.RWMutex),
		copies:   make(map[int][]byte),
		locks:    make(map[int]*sync.RWMutex),
		vm:       vm,
		mpt:      make(map[int]minipage),
		log:      make(map[int]int),
	}
	return &m
}

func NewUpdatedManager(vm memory.VirtualMemory, lm treadmarks.LockManager, bm treadmarks.BarrierManager) *Manager {
	m := Manager{
		copyLock:       new(sync.RWMutex),
		copies:         make(map[int][]byte),
		locks:          make(map[int]*sync.RWMutex),
		vm:             vm,
		mpt:            make(map[int]minipage),
		log:            make(map[int]int),
		LockManager:    lm,
		BarrierManager: bm,
		Mutex:          new(sync.Mutex),
	}
	return &m
}

func (m *Manager) getCopies(pageNr int) []byte {
	m.copyLock.RLock()
	defer m.copyLock.RUnlock()
	return m.copies[pageNr]
}

func (m *Manager) setCopies(pageNr int, val []byte) {
	m.copyLock.Lock()
	defer m.copyLock.Unlock()
	m.copies[pageNr] = val
}

func (m *Manager) deleteCopies(pageNr int) {
	m.copyLock.Lock()
	defer m.copyLock.Unlock()
	delete(m.copies, pageNr)
}

func (m *Manager) doForAllCopies(f func(key int, bytes []byte)) {
	m.copyLock.Lock()
	defer m.copyLock.Unlock()
	for i, bytes := range m.copies {
		f(i, bytes)
	}
}

func (m *Manager) Connect(address string) {
	m.cl = network.NewClient(
		func(message network.Message) error {
			var msg network.MultiviewMessage
			switch message.(type) {
			case network.SimpleMessage:
				msg = network.MultiviewMessage{From: message.GetFrom(), To: message.GetTo(), Type: message.GetType()}
			case network.MultiviewMessage:
				msg = message.(network.MultiviewMessage)
			}
			go m.HandleMessage(msg)
			return nil
		})
	m.cl.Connect(address)
	m.tr = m.cl.GetTransciever()
}

// This is the function to call, when a manager has to handle any message.
// This will call the correct functions, depending on the message type, and
// then send whatever messages needs to be sent afterwards.
func (m *Manager) HandleMessage(message network.MultiviewMessage) {
	log.Println("Manager got message ", message)
	switch t := message.Type; t {

	case READ_REQUEST:
		go m.HandleReadReq(message)

	case WRITE_REQUEST:
		go m.HandleWriteReq(message)

	case INVALIDATE_REPLY:
		m.HandleInvalidateReply(message)

	case MALLOC_REQUEST:
		m.HandleAlloc(message)

	case FREE_REQUEST:
		m.HandleFree(message)
	case WRITE_ACK:
		m.HandleWriteAck(message)
	case READ_ACK:
		m.HandleReadAck(message)
	case LOCK_ACQUIRE_REQUEST:
		m.handleLockAcquireRequest(&message)
	case BARRIER_REQUEST:
		m.handleBarrierRequest(&message)
	case LOCK_RELEASE:
		m.handleLockReleaseRequest(&message)
	}
}

// This translates a message, by adding more information to it. This is information
// that only the manager knows, but which is important for the hosts.
func (m *Manager) translate(message *network.MultiviewMessage) int {
	m.Lock()
	vpage := message.Fault_addr / m.vm.GetPageSize()
	if _, ok := m.mpt[vpage]; ok == false {
		log.Println("vpages in manager before crash:", m.mpt)
		panic(fmt.Errorf("Vpage[%v] did not exist.", vpage))
		return 0
	}
	message.Minipage_base = m.vm.GetPageAddr(message.Fault_addr) + m.mpt[vpage].offset
	message.Minipage_size = m.mpt[vpage].length
	message.Privbase = message.Minipage_base % m.vm.Size()
	m.Unlock()
	return vpage
}

// This handles read requests.
func (m *Manager) HandleReadReq(message network.MultiviewMessage) {
	vpage := m.translate(&message)
	m.locks[vpage].RLock()
	p := m.getCopies(vpage)[0]
	message.To = p
	m.tr.Send(message)
}

// This handles write requests.
func (m *Manager) HandleWriteReq(message network.MultiviewMessage) {
	vpage := m.translate(&message)
	log.Println("translated", message.Fault_addr, " to pageNr", vpage)
	m.locks[vpage].Lock()
	message.Type = INVALIDATE_REQUEST
	if len(m.getCopies(vpage)) < 1 {
		panic("Empty copyset on write request! at vpage" + string(vpage))
	}
	for _, p := range m.getCopies(vpage) {
		message.To = p
		log.Println("Manager sending invalidate to ", p)
		m.tr.Send(message)
	}
}

func (m *Manager) HandleInvalidateReply(message network.MultiviewMessage) {
	vpage := m.translate(&message)

	if len(m.getCopies(vpage)) == 1 {
		message.Type = WRITE_REQUEST
		message.To = m.getCopies(vpage)[0]
		m.tr.Send(message)
		//m.setCopies(vpage, []byte{})
	} else {
		m.setCopies(vpage, m.getCopies(vpage)[1:])
	}
}

func (m *Manager) HandleReadAck(message network.MultiviewMessage) {
	vpage := m.handleAck(message)
	m.locks[vpage].RUnlock()
}

func (m *Manager) HandleWriteAck(message network.MultiviewMessage) {
	vpage := m.handleAck(message)
	m.locks[vpage].Unlock()
}

func (m *Manager) handleAck(message network.MultiviewMessage) int {
	vpage := m.translate(&message)
	alreadyHas := false
	for _, c := range m.getCopies(vpage) {
		if c == message.From {
			alreadyHas = true
			break
		}
	}
	if !alreadyHas {
		m.setCopies(vpage, append(m.getCopies(vpage), message.From))
	}
	return vpage
}

func (m *Manager) HandleAlloc(message network.MultiviewMessage) {
	m.Lock()

	size := message.Minipage_size
	ptr, err := m.vm.Malloc(size)
	panicOnErr(err)
	//generate minipages
	sizeLeft := size
	i := ptr
	resultArray := make([]minipage, 0)
	for sizeLeft > 0 {

		offset := i - m.vm.GetPageAddr(i)
		length := Min(sizeLeft, m.vm.GetPageSize()-offset)
		i = i + length
		sizeLeft = sizeLeft - length
		resultArray = append(resultArray, minipage{offset, length})
	}

	startpg := ptr / m.vm.GetPageSize()
	endpg := (ptr + size) / m.vm.GetPageSize()
	npages := m.vm.Size() / m.vm.GetPageSize()
	//loop over views to find free space
	for i := 1; i < m.vm.GetPageSize(); i++ {
		failed := false
		startpg = startpg + npages
		endpg = endpg + npages
		for j := startpg; j <= endpg; j++ {
			_, exists := m.mpt[j]
			if exists {
				failed = true
				break
			}
		}
		if failed == false {
			break
		}
	}

	//insert into virtual memory
	for i, mp := range resultArray {
		m.mpt[startpg+i] = mp
		m.log[startpg+i] = startpg
		m.locks[startpg+i] = new(sync.RWMutex)
		m.setCopies(startpg+i, []byte{message.From})
	}

	//Send reply to alloc requester
	message.To = message.From
	message.From = 0
	message.Fault_addr = startpg*m.vm.GetPageSize() + m.mpt[startpg].offset
	message.Type = MALLOC_REPLY

	m.Unlock()
	m.tr.Send(message)
}

func (m *Manager) HandleFree(message network.MultiviewMessage) {
	vpage := m.translate(&message)

	//Then we loop over vpages from that vpage. If they point back to this vpage, we free them.
	for i := vpage; true; i++ {
		if m.log[i] != vpage {
			break
		}
		m.locks[i].Lock()
		delete(m.log, i)
		delete(m.mpt, i)
		m.deleteCopies(i)
		delete(m.locks, i)
	}
	m.vm.Free(message.Fault_addr % m.vm.Size())
	message.Type = FREE_REPLY
	message.To = message.From
	m.tr.Send(message)
}

func (m *Manager) handleLockAcquireRequest(message *network.MultiviewMessage) {
	id := message.Id
	m.HandleLockAcquire(id)
	message.From, message.To = message.To, message.From
	message.From = byte(0)
	message.Type = LOCK_ACQUIRE_RESPONSE
	m.tr.Send(message)
}

func (m *Manager) handleLockReleaseRequest(message *network.MultiviewMessage) error {
	id := message.Id
	return m.HandleLockRelease(id, message.From)
}

func (m *Manager) handleBarrierRequest(message *network.MultiviewMessage) {
	id := message.Id
	log.Println("process", message.From, "arrived at barrier", id)
	m.HandleBarrier(id, func() {})
	//barrier over

	message.From, message.To = message.To, message.From
	message.Type = BARRIER_RESPONSE
	m.tr.Send(message)
}

// Here is some utility stuff
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
