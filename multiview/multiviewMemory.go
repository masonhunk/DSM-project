package multiview

import (
	"DSM-project/memory"
	"errors"
)

// implements the VirtualMemory interface and is also a decorator/proxy for it
type MVMem struct {
	vm memory.VirtualMemory
	addrToPage map[int]*vPage
}

type vPage struct {
	pageAddr int
	offset, length int
	access_right byte
}

const NO_ACCESS byte = 0
const READ_ONLY byte = 1
const READ_WRITE byte = 2


func NewMVMem(memSize int, pageByteSize int) *MVMem {
	/*m := new(MVMem)
	m.Stack = make([]byte, memSize)
	m.AccessMap = make(map[int]byte)
	m.PAGE_BYTESIZE = pageByteSize
	m.FreeMemObjects = make([]AddrPair, 1)
	m.FreeMemObjects[0] = AddrPair{0, memSize - 1}
	return m*/
}

func (m *MVMem) Read(addr int) (byte, error) {
	vp := m.addrToPage[addr]
	if vp.access_right == 0 {
		return 0, errors.New("access denied")

	}
	return m.vm.Read(vp.pageAddr + vp.offset)
}

// reads a variable size, up to the size of the minipage
func (m *MVMem) ReadBytes(addr, length int) ([]byte, error) {
	vp := m.addrToPage[addr]
	if length > vp.length {
		return nil, errors.New("overread: tried to read more than contained in the vpage")
	}
	if vp.access_right == 0 {
		return nil, errors.New("access denied")
	}
	return m.vm.ReadBytes(vp.pageAddr + vp.offset, length)
}

func (m *MVMem) Write(addr int, val byte) error {
	vp := m.addrToPage[addr]
	if vp.access_right == 0 {
		return errors.New("access denied")

	}
	return m.vm.Write(vp.pageAddr + vp.offset, val)
}

func (m *MVMem) GetRights(addr int) byte {
	return m.addrToPage[addr].access_right
}

func (m *MVMem) SetRights(addr int, access byte) {
	m.addrToPage[addr].access_right = access
}

func (m *MVMem) GetPageAddr(addr int) int {
	panic("implement me")
}

func (m *MVMem) Malloc(sizeInBytes int) (int, error) {
	panic("implement me")
}

func (m *MVMem) Free(offset, sizeInBytes int) error {
	panic("implement me")
}

