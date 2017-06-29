package memory

import "errors"

type Page struct {
	AccessRight int
	memory []byte
}


// NO_ACCESS is nil
const READ_ONLY byte = 1
const READ_WRITE byte = 2


type VirtualMemory interface {
	Read(addr int) (byte, error)
	Write(addr int, val byte) error
	GetRights(addr int) byte
	SetRights(addr int, access byte)
}

type Vmem struct {
	Stack     []byte
	AccessMap map[int]byte
	PAGE_BYTESIZE int
}

func NewVmem(memSize int, pageByteSize int) *Vmem {
	m := new(Vmem)
	m.Stack = make([]byte, memSize)
	m.AccessMap = make(map[int]byte)
	m.PAGE_BYTESIZE = pageByteSize
	return m
}

func (m *Vmem) Read(addr int) (byte, error) {
	access := m.AccessMap[m.getPageAddr(addr)]
	switch access {
	case 0:
		return 127, errors.New("access denied")
	case READ_ONLY:
		return m.Stack[addr], nil
	case READ_WRITE:
		return m.Stack[addr], nil
	}
	return 127, errors.New("unknown error")
}

func (m *Vmem) Write(addr int, val byte) error {
	access := m.AccessMap[m.getPageAddr(addr)]
	switch access {
	case 0:
		return errors.New("access denied")
	case READ_ONLY:
		return errors.New("access denied")
	case READ_WRITE:
		m.Stack[addr] = val
		return nil
	}
	return errors.New("unknown error")

}

func (m *Vmem) GetRights(addr int) byte {
	return m.AccessMap[m.getPageAddr(addr)]
}

func (m *Vmem) SetRights(addr int, access byte) {
	m.AccessMap[m.getPageAddr(addr)] = access
}


func (m *Vmem) getPageAddr(addr int) int {
	return addr - addr % m.PAGE_BYTESIZE
}