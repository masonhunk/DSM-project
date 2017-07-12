package multiview

import (
	"DSM-project/network"
	"sync"
	"DSM-project/memory"
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

}

func NewManager(tr network.ITransciever, vm memory.VirtualMemory) Manager{
	m := Manager{
		CopySet{copies: make(map[int][]byte),
		locks: make(map[int]*sync.RWMutex)},
		tr,
		vm,
		make(map[int]minipage)}
	return m
}

func (m *Manager) HandleMessage(message network.Message){

}

func (m *Manager) translate(message network.Message) network.Message{
	vpage :=  message.Fault_addr / m.vm.GetPageSize()
	message.Minipage_base = m.vm.GetPageAddr(message.Fault_addr) + m.mpt[vpage].offset
	message.Minipage_size = m.mpt[vpage].length
	message.Privbase = message.Minipage_base % m.vm.Size()
	return message
}

func (m *Manager) HandleReadReq(message network.Message){
	message = m.translate(message)
	vpage :=  message.Fault_addr / m.vm.GetPageSize()
	m.cs.locks[vpage].RLock()
	p := m.cs.copies[vpage][0]
	message.To = p
	m.tr.Send(message)
}

func (m *Manager) HandleWriteReq(message network.Message){
	message = m.translate(message)
	vpage :=  message.Fault_addr / m.vm.GetPageSize()
	m.cs.locks[vpage].Lock()
	for _, p := range m.cs.copies[vpage]{
		message.To = p
		message.Type = INVALIDATE_REQUEST
		m.tr.Send(message)
	}
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

func (m *Manager) HandleAlloc(message network.Message){
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
		startpg = startpg + m.vm.GetPageSize()
		endpg = endpg + m.vm.GetPageSize()
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
	}

	message.To=message.From


}

func (m *Manager) HandleFree(){}