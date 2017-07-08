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
	FreeVPages []interval
}

type interval struct {
	start, end int
}

type VPage struct {
	PageAddr       int
	Offset, Length int
	Access_right   byte
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
	m.FreeVPages = make([]interval, 0)
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
	sizeLeft := sizeInBytes
	i := ptr
	resultArray := make([]VPage, 0)
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
		resultArray = append(resultArray, vp)
		sizeLeft -= length
		i = nextPageAddr
	}

	//find suitable free spot in vPage array to insert. Otherwise just append
	for i, interval := range m.FreeVPages {
		if interval.end - interval.start >= len(resultArray) {
			//insert into free interval
			m.AddrToPage = append(m.AddrToPage[:interval.start], resultArray...)
			if interval.end - interval.start == len(resultArray) {
				//remove interval in freevpages list
				m.FreeVPages = append(m.FreeVPages[:i], m.FreeVPages[i+1:]...)
			} else {
				m.FreeVPages[i].start += len(resultArray)
			}
			return (interval.start * m.VPAGE_SIZE) + m.AddrToPage[interval.start].Offset, nil
		}
	}
	//else if no free spots, append
	resultPtr := len(m.AddrToPage) * m.VPAGE_SIZE + resultArray[0].Offset
	m.AddrToPage = append(m.AddrToPage, resultArray...)
	return resultPtr, nil
}

func (m *MVMem) Free(offset, sizeInBytes int) error {
	if offset + sizeInBytes >= m.getLastAddr() || offset < 0 || offset + sizeInBytes < 0 {
		return errors.New("index out of bounds")
	}
	index := m.GetPageAddr(offset)/m.GetPageSize()
	err := m.vm.Free(m.AddrToPage[index].Offset, sizeInBytes)
	if err != nil {
		return err
	}
	sizeLeft := sizeInBytes
	resultArray := make([]VPage, 0)
	for sizeLeft > 0 {
		if index >= len(m.AddrToPage) {
			//stop if trying to free more memory than available. Doesn't throw an error
			break
		}
		vPage := m.AddrToPage[index]
		if vPage.Length > sizeLeft {
			//if last vPage to be removed has > size than size left, shrink it.
			vPage.Length -= sizeLeft
			break
		} else {
			//"remove" startvPage from list, ie. set to nil and add to free vpage interval list

			m.AddrToPage = append(m.AddrToPage[:index], m.AddrToPage[index + 1:]...)

			sizeLeft -= vPage.Length

		}
		index++
	}
	return nil
}


func (m *MVMem) GetPageSize() int {
	return m.VPAGE_SIZE
}

// takes an address in the vPage memory space and translates it into the corresponding address in the memory
func (m *MVMem) vPageAddrToMemoryAddr(addr int) (int, error) {
	vp := m.AddrToPage[m.GetPageAddr(addr)/m.GetPageSize()]
	return vp.PageAddr + (addr - m.GetPageAddr(addr)), nil
}


func (m *MVMem) Size() int {
	return m.vm.Size()
}


func (m *MVMem) getLastAddr() int {
	lastVP := m.AddrToPage[len(m.AddrToPage) - 1]
	return len(m.AddrToPage)*m.VPAGE_SIZE - m.VPAGE_SIZE + lastVP.Length - 1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}


func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}