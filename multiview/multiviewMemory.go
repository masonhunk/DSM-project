package multiview

import (
	"DSM-project/memory"
	"errors"
)

// implements the VirtualMemory interface and is also a decorator/proxy for it
type MVMem struct {
	vm memory.VirtualMemory
	addrToPage []*vPage
	VPAGE_SIZE int
}


type vPage struct {
	pageAddr int
	offset, length int
	access_right byte
}

const NO_ACCESS byte = 0
const READ_ONLY byte = 1
const READ_WRITE byte = 2


func NewMVMem(virtualMemory memory.VirtualMemory) *MVMem {
	m := new(MVMem)
	m.addrToPage = make([]*vPage, 100)
	m.vm = virtualMemory
	m.VPAGE_SIZE = virtualMemory.GetPageSize()
	return m
}

func (m *MVMem) Read(addr int) (byte, error) {
	vp := m.addrToPage[m.GetPageAddr(addr)]
	if vp.access_right == 0 {
		return 0, errors.New("access denied")

	}
	return m.vm.Read(vp.pageAddr + vp.offset)
}

// reads a variable size, up to the size of the minipage
func (m *MVMem) ReadBytes(addr, length int) ([]byte, error) {
	vp := m.addrToPage[m.GetPageAddr(addr)]
	if length > vp.length {
		return nil, errors.New("overread: tried to read more than contained in the vpage")
	}
	if vp.access_right == 0 {
		return nil, errors.New("access denied")
	}
	return m.vm.ReadBytes(vp.pageAddr + vp.offset, length)
}

func (m *MVMem) Write(addr int, val byte) error {
	vp := m.addrToPage[m.GetPageAddr(addr)]
	if vp.access_right == 0 {
		return errors.New("access denied")

	}
	return m.vm.Write(vp.pageAddr + vp.offset, val)
}

func (m *MVMem) GetRights(addr int) byte {
	return m.addrToPage[m.GetPageAddr(addr)].access_right
}

func (m *MVMem) SetRights(addr int, access byte) {
	m.addrToPage[m.GetPageAddr(addr)].access_right = access
}

func (m *MVMem) GetPageAddr(addr int) int {
	return addr - addr % m.VPAGE_SIZE
}

func (m *MVMem) Malloc(sizeInBytes int) (int, error) {
	ptr, err := m.vm.Malloc(sizeInBytes)
	if err != nil {
		return 0, err
	}
	resultPtr := -1
	sizeLeft := sizeInBytes
	i := ptr
	for sizeLeft > 0 {
		nextPageAddr := i + m.vm.GetPageSize() - i % m.vm.GetPageSize()
		length := 0
		if nextPageAddr - i < sizeLeft {
			length = sizeLeft
		} else {
			length = nextPageAddr - i
		}
		vp := vPage {
			pageAddr: m.vm.GetPageAddr(i),
			offset: 	i - m.vm.GetPageAddr(i),
			length: 	length,
			access_right: READ_WRITE,
		}
		m.addrToPage = append(m.addrToPage, &vp)
		if resultPtr == -1 {
			resultPtr = len(m.addrToPage) - 1 * m.VPAGE_SIZE
		}
		sizeLeft -= length
		i = nextPageAddr
	}
	return resultPtr, nil
}

func (m *MVMem) Free(offset, sizeInBytes int) error {
	panic("implement me")
}


func (m *MVMem) GetPageSize() int {
	return m.VPAGE_SIZE
}
