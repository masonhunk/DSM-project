package treadmarks

import (
	"DSM-project/network"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
)

var _ = fmt.Print //TODO: remove when done

//Interfaces
type LockManager interface {
	HandleLockAcquire(id int) byte
	HandleLockRelease(id int, newOwner byte) error
}

type BarrierManager interface {
	HandleBarrier(id int, f func()) *sync.WaitGroup
}

//Lock manager implementation
type LockManagerImp struct {
	locks map[int]*sync.Mutex
	last  map[int]byte
	*sync.Mutex
}

func NewLockManagerImp() *LockManagerImp {
	lm := new(LockManagerImp)
	lm.locks = make(map[int]*sync.Mutex)
	lm.Mutex = new(sync.Mutex)
	lm.last = make(map[int]byte)
	return lm
}

func (lm *LockManagerImp) HandleLockAcquire(id int) byte {
	lm.Lock()
	lock, ok := lm.locks[id]
	if ok == false {
		lock = new(sync.Mutex)
		lm.locks[id] = lock
	}
	lm.Unlock()
	lock.Lock()
	return lm.last[id]
}

func (lm *LockManagerImp) HandleLockRelease(id int, newOwner byte) error {
	lm.Lock()
	lock, ok := lm.locks[id]
	if ok == false {
		return errors.New("LockManager doesn't have a lock with ID " + strconv.Itoa(id))
	}
	lm.Unlock()
	lm.last[id] = newOwner
	lock.Unlock()
	return nil
}

//Barrier manager implementation
type BarrierManagerImp struct {
	barriers map[int]*sync.WaitGroup
	nodes    int
	*sync.Mutex
}

func NewBarrierManagerImp(nodes int) *BarrierManagerImp {
	bm := new(BarrierManagerImp)
	bm.nodes = nodes
	bm.barriers = make(map[int]*sync.WaitGroup)
	bm.Mutex = new(sync.Mutex)
	return bm
}

func (bm *BarrierManagerImp) HandleBarrier(id int, f func()) *sync.WaitGroup {
	bm.Lock()
	barrier, ok := bm.barriers[id]
	if ok == false {
		barrier = new(sync.WaitGroup)
		barrier.Add(bm.nodes)
		bm.barriers[id] = barrier
	}
	bm.Unlock()
	f()
	barrier.Done()
	barrier.Wait()
	return barrier
}

type tm_Manager struct {
	vc   Vectorclock
	myId byte
	BarrierManager
	LockManager
	network.ITransciever //embedded type
	nodes                int
	tm                   *TreadMarks //the host instance on which this manager runs

}

func NewTest_TM_Manager(tr network.ITransciever, bm BarrierManager, lm LockManager, nodes int, tm *TreadMarks) *tm_Manager {
	m := new(tm_Manager)
	m.BarrierManager = bm
	m.LockManager = lm
	m.ITransciever = tr
	m.nodes = nodes
	m.myId = byte(0)
	m.tm = tm

	return m
}

func NewTM_Manager(conn net.Conn, bm BarrierManager, lm LockManager, tm *TreadMarks) *tm_Manager {
	m := new(tm_Manager)
	m.BarrierManager = bm
	m.LockManager = lm
	m.ITransciever = network.NewTransciever(conn, func(message network.Message) error {
		return m.HandleMessage(message)
	})
	m.vc = *NewVectorclock(tm.nrProcs)
	m.nodes = tm.nrProcs
	m.myId = byte(0)
	m.tm = tm

	return m
}

func (m *tm_Manager) HandleMessage(message network.Message) error {
	msg, ok := message.(TM_Message)
	if ok == false {
		if _, ok := message.(network.SimpleMessage); !ok {
			panic("Message could not be converted.")
		} else {
			return nil
		}
	}
	response := new(TM_Message)
	var err error
	switch msg.Type {
	case LOCK_ACQUIRE_REQUEST:
		response = m.handleLockAcquireRequest(&msg)
	case LOCK_RELEASE:
		err = m.handleLockReleaseRequest(&msg)
	case BARRIER_REQUEST:
		response = m.handleBarrierRequest(&msg)
	case MALLOC_REQUEST:
		panic("Implement me!")
	case FREE_REQUEST:
		panic("Implement me!")
	default:
		panic("I saw a message I was not allowed to see!")
	}

	if response.Type != "" {
		m.ITransciever.Send(*response)
	}
	return err
}

func (m *tm_Manager) handleLockAcquireRequest(message *TM_Message) *TM_Message {
	id := message.Id
	lastOwner := m.HandleLockAcquire(id)
	message.To = lastOwner
	if lastOwner == 0 {
		message.To = message.From
		message.From = m.myId
		message.Type = LOCK_ACQUIRE_RESPONSE
	}
	return message
}

func (m *tm_Manager) handleLockReleaseRequest(message *TM_Message) error {
	id := message.Id
	return m.HandleLockRelease(id, message.From)
}

func (m *tm_Manager) handleBarrierRequest(message *TM_Message) *TM_Message {
	var msg TM_Message = *message
	id := message.Id
	var currTick uint
	m.HandleBarrier(id, func() {
		if m.tm != nil {
			m.tm.incorporateIntervalsIntoDatastructures(message)
			m.vc = *m.vc.Merge(message.VC)
			currTick = m.vc.GetTick(byte(0))
		}
	})
	//barrier over

	if m.tm != nil {
		msg.Intervals = m.tm.GetAllUnseenIntervals(msg.VC)

	}
	m.vc.SetTick(byte(0), currTick+1)
	msg.From, msg.To = msg.To, msg.From
	msg.VC = m.vc
	msg.Type = BARRIER_RESPONSE
	return &msg
}
