package multiview

import (
	"DSM-project/network"
	"sync"
	"DSM-project/memory"
	"fmt"
	"errors"
)

//Each minipage consists of an offset and a length.
type minipage struct {
	offset, length int
}

//this is the actual manager.
type Manager struct{
	tr network.ITransciever
	cl *network.Client 		//The transciever that we are sending messages over.
	vm memory.VirtualMemory 		//The virtual memory object we are working on in the system.
	mpt map[int]minipage			//Minipagetable
	log map[int]int 				//A map, where each entrance points to
									// the first vpage of this allocation. Used for freeing.
	copies map[int][]byte			//A map of who has copies of what vpage
	locks map[int]*sync.RWMutex		//A map of locks belonging to each vpage.
}

// Returns the pointer to a manager object.
func NewManager(vm memory.VirtualMemory) *Manager{
	//TODO remove the line below
	fmt.Println("") //Just to make to compiler be quiet for now.
	m := Manager{
		copies: make(map[int][]byte),
		locks: make(map[int]*sync.RWMutex),
		vm: vm,
		mpt: make(map[int]minipage),
		log: make(map[int]int),
	}
	return &m
}

func(m *Manager) Connect(address string) {
	m.cl = network.NewClient(func(message network.Message)error{go m.HandleMessage(message); return nil})
	m.cl.Connect(address)
	m.tr = m.cl.GetTransciever()
}

// This is the function to call, when a manager has to handle any message.
// This will call the correct functions, depending on the message type, and
// then send whatever messages needs to be sent afterwards.
func (m *Manager) HandleMessage(message network.Message) error {
	fmt.Println("Manager got message ", message)
	switch t := message.Type; t{

	case READ_REQUEST:
		message, err :=m.HandleReadReq(message)
		if err != nil{return err}
		m.tr.Send(message)

	case WRITE_REQUEST:
		message, err :=m.HandleWriteReq(message)
		if err != nil{return err}
		for _, mes := range message{m.tr.Send(mes)}

	case INVALIDATE_REPLY:
		err := m.HandleInvalidateReply(message)
		if err != nil{return err}

	case MALLOC_REQUEST:
		message, err :=m.HandleAlloc(message)
		if err != nil{return err}
		m.tr.Send(message)

	case FREE_REQUEST:
		message, err :=m.HandleFree(message)
		if err != nil{return err}
		m.tr.Send(message)
	}
	return nil
}

// This translates a message, by adding more information to it. This is information
// that only the manager knows, but which is important for the hosts.
func (m *Manager) translate(message network.Message) (network.Message, error) {
	vpage :=  message.Fault_addr / m.vm.GetPageSize()
	if _, ok:= m.mpt[vpage]; ok == false{
		message.Err = errors.New("vpage did not exist.")
		return message, errors.New("vpage did not exist.")
	}
	message.Minipage_base = m.vm.GetPageAddr(message.Fault_addr) + m.mpt[vpage].offset
	message.Minipage_size = m.mpt[vpage].length
	message.Privbase = message.Minipage_base % m.vm.Size()
	return message, nil
}

// This handles read requests.
func (m *Manager) HandleReadReq(message network.Message) (network.Message, error){
	var err error
	message, err = m.translate(message)
	vpage :=  message.Fault_addr / m.vm.GetPageSize()

	if _, ok := m.mpt[vpage]; !ok { //If the page doesnt exist, we return nil and an error.
		return network.Message{}, errors.New("vpage have not been allocated.")
	}

	m.locks[vpage].RLock()
	p := m.copies[vpage][0]
	message.To = p
	return message, err
}

// This handles write requests.
func (m *Manager) HandleWriteReq(message network.Message) ([]network.Message, error){
	var err error
	message, err = m.translate(message)
	vpage :=  message.Fault_addr / m.vm.GetPageSize()

	if _, ok := m.mpt[vpage]; !ok { //If the page doesnt exist, we return nil and an error.
		return nil, errors.New("vpage have not been allocated.")
	}

	m.locks[vpage].Lock()
	message.Type = INVALIDATE_REQUEST
	messages := []network.Message{}
	fmt.Println("Copies contained ", m.copies)

	for _, p := range m.copies[vpage]{
		message.To = p
		fmt.Println("Sending invalidate to ", p)
		messages = append(messages, message)
	}
	return messages, err
}

func (m *Manager) HandleInvalidateReply(message network.Message) error{
	vpage :=  message.Fault_addr / m.vm.GetPageSize()
	c :=m.copies[vpage]
	fmt.Println("Length of c is ", len(c))
	if len(c) == 1{
		message.Type = WRITE_REQUEST
		message.To = c[0]
		m.tr.Send(message)
		c = []byte{}
	} else {
		c = c[1:]
	}
	return nil
}

func (m *Manager) HandleReadAck(message network.Message) error{
	vpage := m.handleAck(message)
	m.locks[vpage].RUnlock()
	return nil
}

func (m *Manager) HandleWriteAck(message network.Message) error{
	vpage := m.handleAck(message)
	m.locks[vpage].Unlock()
	return nil
}


func (m *Manager) handleAck(message network.Message) int{
	vpage := m.vm.GetPageAddr(message.Fault_addr) / m.vm.GetPageSize()
	m.copies[vpage]=append(m.copies[vpage], message.From)
	return vpage
}

func (m *Manager) HandleAlloc(message network.Message) (network.Message, error){

	size := message.Minipage_size
	ptr, _:= m.vm.Malloc(size)

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

	startpg := ptr/m.vm.GetPageSize()
	endpg := (ptr+size)/m.vm.GetPageSize()
	npages := m.vm.Size()/m.vm.GetPageSize()
	//loop over views to find free space
	for i := 1; i < m.vm.GetPageSize(); i++ {
		failed := false
		startpg = startpg + npages
		endpg = endpg + npages
		for j := startpg; j <= endpg; j++  {
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
		m.mpt[startpg + i] = mp
		m.log[startpg + i] = startpg
		m.locks[startpg+i] = new(sync.RWMutex)
		m.copies[startpg+i] = []byte{message.From}
	}

	//Send reply to alloc requester
	message.To=message.From
	message.From = 1
	message.Fault_addr = startpg*m.vm.GetPageSize() + m.mpt[startpg].offset
	message.Type = MALLOC_REPLY

	return message, nil

}

func (m *Manager) HandleFree(message network.Message) (network.Message, error){
	var err error
	message, err = m.translate(message)

	//First we get the first vpage the allocation is in
	vpage := m.vm.GetPageAddr(message.Fault_addr) / m.vm.GetPageSize()


	//Then we loop over vpages from that vpage. If they point back to this vpage, we free them.
	for i := vpage; true ; i++{
		if m.log[i] != vpage{
			break
		}
		m.locks[i].Lock()
		delete(m.log, i)
		delete(m.mpt, i)
		delete(m.copies, i)
		delete(m.locks,i)
	}
	m.vm.Free(message.Fault_addr % m.vm.Size())
	message.Type = FREE_REPLY
	message.To = message.From
	return message, err
}


// Here is some utility stuff
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
/*
func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}*/