package multiview

import (
	"DSM-project/network"
	"sync"
	"DSM-project/memory"
	"fmt"
	"errors"
)

type CopySet struct{
	copies map[int][]byte
	locks map[int]*sync.RWMutex
}

type minipage struct {
	offset, length int
}


type Manager struct{
	cs CopySet
	tr network.ITransciever
	vm memory.VirtualMemory
	mpt map[int]minipage
	log map[int]int //A map, where each entrance points to the first vpage of this allocation. Used for freeing.
}

func NewManager(tr network.ITransciever, vm memory.VirtualMemory) Manager{
	//TODO remove the line below
	fmt.Println("") //Just to make to compiler be quiet for now.
	m := Manager{
		CopySet{copies: make(map[int][]byte),
		locks: make(map[int]*sync.RWMutex)},
		tr,
		vm,
		make(map[int]minipage),
		make(map[int]int),
	}
	return m
}

func (m *Manager) HandleMessage(message network.Message){
	switch t := message.Type; t{
	case READ_REQUEST:
		message, _ :=m.HandleReadReq(message)
		m.tr.Send(message)
	case WRITE_REQUEST:
		message, _ :=m.HandleWriteReq(message)
		for _, mes := range message{
			m.tr.Send(mes)
		}
	case INVALIDATE_REPLY:
		m.HandleInvalidateReply(message)

	case MALLOC_REQUEST:
		message, _ :=m.HandleAlloc(message)
		m.tr.Send(message)
	case FREE_REQUEST:
		message, _ :=m.HandleFree(message)
		m.tr.Send(message)
	}
}

func (m *Manager) translate(message network.Message) network.Message{
	vpage :=  m.vm.GetPageAddr(message.Fault_addr)/m.vm.GetPageSize()
	message.Minipage_base = m.vm.GetPageAddr(message.Fault_addr) + m.mpt[vpage].offset
	message.Minipage_size = m.mpt[vpage].length
	message.Privbase = message.Minipage_base % m.vm.Size()
	return message
}

func (m *Manager) HandleReadReq(message network.Message) (network.Message, error){
	message = m.translate(message)
	vpage :=  message.Fault_addr / m.vm.GetPageSize()
	m.cs.locks[vpage].RLock()
	p := m.cs.copies[vpage][0]
	message.To = p
	m.tr.Send(message)
	return message, nil
}

func (m *Manager) HandleWriteReq(message network.Message) ([]network.Message, error){
	message = m.translate(message)
	vpage :=  message.Fault_addr / m.vm.GetPageSize()

	if _, ok := m.mpt[vpage]; !ok {
		return nil, errors.New("vpage have not been allocated.")
	}
	m.cs.locks[vpage].Lock()
	message.Type = INVALIDATE_REQUEST
	messages := []network.Message{}

	for _, p := range m.cs.copies[vpage]{
		message.To = p

		messages = append(messages, message)
	}
	return messages, nil
}

func (m *Manager) HandleInvalidateReply(message network.Message){
	vpage :=  message.Fault_addr / m.vm.GetPageSize()
	c :=m.cs.copies[vpage]
	if len(c) == 1{
		message.Type = WRITE_REQUEST
		message.To = c[0]
		m.tr.Send(message)
		c = []byte{}
	} else {
		c = c[1:]
	}
}

func (m *Manager) HandleReadAck(message network.Message){
	vpage := m.handleAck(message)
	m.cs.locks[vpage].RUnlock()
}

func (m *Manager) HandleWriteAck(message network.Message){
	vpage := m.handleAck(message)
	m.cs.locks[vpage].Unlock()
}


func (m *Manager) handleAck(message network.Message) int{
	vpage :=  message.Fault_addr / m.vm.GetPageSize()
	m.cs.copies[vpage]=append(m.cs.copies[vpage], message.From)
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
		nextPageAddr := i + m.vm.GetPageSize() - (i % m.vm.GetPageSize())
		length := 0
		if nextPageAddr - i > sizeLeft {
			length = sizeLeft
		} else {
			length = nextPageAddr - i
		}
		mp := minipage{
			offset: i - m.vm.GetPageAddr(i),
			length: length,
		}
		resultArray = append(resultArray, mp)
		sizeLeft -= length
		i = nextPageAddr

	}

	startpg := ptr/m.vm.GetPageSize()
	endpg := (ptr+size)/m.vm.GetPageSize()

	//loop over views to find free space
	for i := 1; i < m.vm.GetPageSize(); i++ {
		failed := false
		startpg = startpg + m.vm.Size()/m.vm.GetPageSize()
		endpg = endpg + m.vm.Size()/m.vm.GetPageSize()
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
		m.cs.locks[startpg+i] = new(sync.RWMutex)
		m.cs.copies[startpg+i] = []byte{message.From}
	}

	//Send reply to alloc requester
	message.To=message.From
	message.Fault_addr = startpg*m.vm.GetPageSize() + m.mpt[startpg].offset
	message.Type = MALLOC_REPLY
	return message, nil

}

func (m *Manager) HandleFree(message network.Message) (network.Message, error){
	message = m.translate(message)

	//First we get the first vpage the allocation is in
	vpage := m.vm.GetPageAddr(message.Fault_addr) / m.vm.GetPageSize()


	//Then we loop over vpages from that vpage. If they point back to this vpage, we free them.
	for i := vpage; true ; i++{
		if m.log[i] != vpage{
			break
		}
		m.cs.locks[i].Lock()
		delete(m.log, i)
		delete(m.mpt, i)
		delete(m.cs.copies, i)
		delete(m.cs.locks,i)
	}
	message.Type = FREE_REPLY
	message.To = message.From
	return message, nil
}