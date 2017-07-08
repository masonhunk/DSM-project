package multiview

import (
	"DSM-project/memory"
	"errors"
)

// implements the VirtualMemory interface and is also a decorator/proxy for it
type MVMem struct {
	vm         memory.VirtualMemory
	AddrToPage []VPage
	VPAGE_SIZE int
	FreeMemObjects []AddrPair
}


type VPage struct {
	PageAddr       int
	Offset, Length int
	Access_right   byte
}

type AddrPair struct {
	Start, End int
}

const NO_ACCESS byte = 0
const READ_ONLY byte = 1
const READ_WRITE byte = 2


func NewMVMem(virtualMemory memory.VirtualMemory) *MVMem {
	m := new(MVMem)
	m.AddrToPage = make([]VPage, 0)
	m.vm = virtualMemory
	m.VPAGE_SIZE = (virtualMemory).GetPageSize()
	memory.NO_ACCESS = 2
	memory.READ_WRITE = 0
	return m
}

func (m *MVMem) Read(addr int) (byte, error) {
	vp := m.AddrToPage[m.GetPageAddr(addr)/m.GetPageSize()]
	if vp.Access_right == 0 {
		return 0, errors.New("access denied")

	}
	res, err := m.vPageAddrToMemoryAddr(addr)
	if err != nil {
		return 0, err
	}
	return m.vm.Read(res)
}

// reads a variable size, up to the size of the minipage
func (m *MVMem) ReadBytes(addr, length int) ([]byte, error) {
	for i := addr; i < addr + length; i += m.VPAGE_SIZE {
		vp := m.AddrToPage[m.GetPageAddr(i)/m.GetPageSize()]
		if vp.Access_right == NO_ACCESS {
			return nil, errors.New("Access Denied")
		}
	}
	vp := m.AddrToPage[m.GetPageAddr(addr)/m.GetPageSize()]
	return m.vm.ReadBytes(addr - vp.PageAddr, length)
}

func (m *MVMem) Write(addr int, val byte) error {
	vp := m.AddrToPage[m.GetPageAddr(addr)/m.GetPageSize()]
	if vp.Access_right == 0 {
		return errors.New("access denied")

	}
	res, err := m.vPageAddrToMemoryAddr(addr)
	if err != nil {
		return err
	}
	return m.vm.Write(res, val)
}

func (m *MVMem) GetRights(addr int) byte {
	return m.AddrToPage[m.GetPageAddr(addr)/m.GetPageSize()].Access_right
}

func (m *MVMem) SetRights(addr int, access byte) {
	m.AddrToPage[m.GetPageAddr(addr)/m.GetPageSize()].Access_right = access
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
		nextPageAddr := i + m.vm.GetPageSize() - (i % m.vm.GetPageSize())
		length := 0
		if nextPageAddr - i > sizeLeft {
			length = sizeLeft
		} else {
			length = nextPageAddr - i
		}
		vp := VPage{
			PageAddr:     m.vm.GetPageAddr(i),
			Offset:       i - m.vm.GetPageAddr(i),
			Length:       length,
			Access_right: READ_WRITE,
		}
		m.AddrToPage = append(m.AddrToPage, vp)
		if resultPtr == -1 {
			resultPtr = (len(m.AddrToPage) - 1) * m.VPAGE_SIZE + vp.Offset
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

// takes an address in the vPage memory space and translates it into the corresponding address in the memory
func (m *MVMem) vPageAddrToMemoryAddr(addr int) (int, error) {
	vp := m.AddrToPage[m.GetPageAddr(addr)/m.GetPageSize()]
	return vp.PageAddr + (addr - m.GetPageAddr(addr)), nil
}
