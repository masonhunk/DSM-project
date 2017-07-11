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
	mallocHistory map[int]int //maps address to length
}

func (m *MVMem) addFaultListener(l memory.FaultListener) {
	panic("implement me")
}

type interval struct {
	start, end int
}

type VPage struct {
	PageAddr       int //the address of the actual page to which this vpage points
	Offset, Length int
	Access_right   byte
}


func NewMVMem(virtualMemory memory.VirtualMemory) *MVMem {
	m := new(MVMem)
	m.AddrToPage = make([]VPage, 0)
	m.vm = virtualMemory
	m.VPAGE_SIZE = (virtualMemory).GetPageSize()
	m.vm.AccessRightsDisabled(true)
	m.FreeVPages = make([]interval, 0)
	m.mallocHistory = make(map[int]int)
	return m
}

func (m *MVMem) Read(addr int) (byte, error) {
	vp := m.AddrToPage[m.GetPageAddr(addr)/m.GetPageSize()]
	if vp.Access_right == memory.NO_ACCESS {
		return 0, errors.New("access denied")

	}
	res, err := m.vPageAddrToMemoryAddr(addr)
	if err != nil {
		return 0, err
	}
	return m.vm.Read(res)
}

//returns all data in the minipage for which the address resides
func (m *MVMem) ReadMinipage(addr int) ([]byte, error) {
	vp := m.AddrToPage[m.GetPageAddr(addr)/m.GetPageSize()]
	return m.ReadBytes(m.GetPageAddr(addr) + vp.Offset, vp.Length)
}

// reads a variable size, up to the size of the minipage
func (m *MVMem) ReadBytes(addr, length int) ([]byte, error) {
	//check access rights
	for i := addr; i < addr + length; i += m.VPAGE_SIZE {
		vp := m.AddrToPage[m.GetPageAddr(i)/m.GetPageSize()]
		if vp.Access_right == memory.NO_ACCESS {
			return nil, errors.New("Access Denied")
		}
	}
	realAddr, _ := m.vPageAddrToMemoryAddr(addr)
	return m.vm.ReadBytes(realAddr, length)
}

func (m *MVMem) Write(addr int, val byte) error {
	vp := m.AddrToPage[m.GetPageAddr(addr)/m.GetPageSize()]
	if vp.Access_right == memory.NO_ACCESS || vp.Access_right == memory.READ_ONLY  {
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

//returns the vPage address for this addr
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
			Access_right: memory.READ_WRITE,
		}
		resultArray = append(resultArray, vp)
		sizeLeft -= length
		i = nextPageAddr
	}

	//find suitable free spot in vPage array to insert. Otherwise just append
	for i, interval := range m.FreeVPages {
		if interval.end - interval.start >= len(resultArray) {
			//insert into free interval
			m.AddrToPage = append(m.AddrToPage[:interval.start], append(resultArray, m.AddrToPage[interval.start:]...)...)
			if interval.end - interval.start == len(resultArray) {
				//remove interval in freevpages list
				m.FreeVPages = append(m.FreeVPages[:i], m.FreeVPages[i+1:]...)
			} else {
				m.FreeVPages[i].start += sizeInBytes
			}
			return (interval.start * m.VPAGE_SIZE) + m.AddrToPage[interval.start].Offset, nil
		}
	}
	//else if no free spots, append
	resultPtr := len(m.AddrToPage) * m.VPAGE_SIZE + resultArray[0].Offset
	m.AddrToPage = append(m.AddrToPage, resultArray...)
	m.mallocHistory[resultPtr] = sizeInBytes
	return resultPtr, nil
}

func (m *MVMem) Free(pointer int) error {
	size := m.mallocHistory[pointer]
	if size == 0 {
		return errors.New("invalid reference: no corresponding malloc found")
	}
	m.mallocHistory[pointer] = 0
	addr, err := m.vPageAddrToMemoryAddr(pointer)
	if err != nil {
		return err
	}
	if err := m.vm.Free(addr); err != nil {
		return err
	}
	start := pointer
	end := start + size - 1
	var newlist []interval
	j := -1
	for i, pair := range m.FreeVPages {
		if pair.end + 1 < start  {
			newlist = append(newlist, pair)
		} else if end + 1 < pair.start {
			j = i
			break
		} else {
			start = min(start, pair.start)
			end = max(end, pair.end)

		}
	}
	newlist = append(newlist, interval{start, end})
	if j != -1 {
		newlist = append(newlist, m.FreeVPages[j:]...)
	}
	m.FreeVPages = newlist


	return nil
}


func (m *MVMem) GetPageSize() int {
	return m.VPAGE_SIZE
}


func (m *MVMem) AccessRightsDisabled(b bool) {
	panic("implement me")
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