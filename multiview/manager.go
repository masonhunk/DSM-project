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

func (m *Manager) translate(){} //Not necessary atm.

func (m *Manager) HandleReadReq(message network.Message){
	i := m.vm.GetPageAddr(message.Fault_addr)
	m.cs.locks[i].RLock()
	p := m.cs.copies[i][0]
	message.To = p
	m.tr.Send(message)
}

func (m *Manager) HandleWriteReq(message network.Message){
	i := m.vm.GetPageAddr(message.Fault_addr)
	m.cs.locks[i].Lock()
	for _, p := range m.cs.copies[i]{
		message.To = p
		message.Type = network.INVALIDATE_REQUEST
		m.tr.Send(message)
	}
}

func (m *Manager) HandleInvalidateReply(message network.Message){
	i := m.vm.GetPageAddr(message.Fault_addr)
	c :=m.cs.copies[i]
	if len(c) == 1{
		message.Type = network.WRITE_REQUEST
		message.To = c[0]
		m.tr.Send(message)
		c = []byte{}
	} else {
		c = c[1:]
	}
}

func (m *Manager) HandleReadAck(message network.Message){
	i := m.handleAck(message)
	m.cs.locks[i].RUnlock()
}

func (m *Manager) HandleWriteAck(message network.Message){
	i := m.handleAck(message)
	m.cs.locks[i].Unlock()
}

func (m *Manager) handleAck(message network.Message) int{
	i := mem.GetPageAddr(message.Fault_addr)
	m.cs.copies[i]=append(m.cs.copies[i], message.From)
	return i
}

func (m *Manager) HandleAlloc(size int){
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


}

func (m *Manager) HandleFree(){}