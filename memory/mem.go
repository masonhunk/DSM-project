package memory

import (
	"errors"
	"strconv"
)

var AccessDeniedErr = errors.New("access denied")

const NO_ACCESS byte = 0
const READ_ONLY byte = 1
const READ_WRITE byte = 2

type FaultListener func(addr int, faultType byte) //faultType = 0 => read fault, = 1 => write fault

type VirtualMemory interface {
	Read(addr int) (byte, error)
	ReadBytes(addr, length int) ([]byte, error)
	Write(addr int, val byte) error
	GetRights(addr int) byte
	SetRights(addr int, access byte)
	GetPageAddr(addr int) int
	Malloc(sizeInBytes int) (int, error)
	Free(pointer int) error
	GetPageSize() int
	Size() int
	AccessRightsDisabled(b bool)
	addFaultListener(l FaultListener)
}

type Vmem struct {
	Stack          []byte
	AccessMap      map[int]byte
	PAGE_BYTESIZE  int
	FreeMemObjects []AddrPair
	mallocHistory  map[int]int //maps address to length
	arDisabled     bool
	faultListeners []FaultListener
}

func (m *Vmem) addFaultListener(l FaultListener) {
	m.faultListeners = append(m.faultListeners, l)
}

func (m *Vmem) AccessRightsDisabled(b bool) {
	m.arDisabled = b
}

func (m *Vmem) Size() int {
	return len(m.Stack)
}

func (m *Vmem) GetPageSize() int {
	return m.PAGE_BYTESIZE
}

func (m *Vmem) Free(pointer int) error {
	size := m.mallocHistory[pointer]
	if size == 0 {
		return errors.New("invalid reference: no corresponding malloc found")
	}
	start := pointer
	end := start + size - 1
	var newlist []AddrPair
	j := -1
	for i, pair := range m.FreeMemObjects {
		if pair.End+1 < start {
			newlist = append(newlist, pair)
		} else if end+1 < pair.Start {
			j = i
			break
		} else {
			start = min(start, pair.Start)
			end = max(end, pair.End)

		}
	}
	newlist = append(newlist, AddrPair{start, end})
	if j != -1 {
		newlist = append(newlist, m.FreeMemObjects[j:]...)
	}
	m.FreeMemObjects = newlist
	return nil
}

type AddrPair struct {
	Start, End int
}

func (m *Vmem) Malloc(sizeInBytes int) (int, error) {
	for i, pair := range m.FreeMemObjects {
		if pair.End-pair.Start+1 >= sizeInBytes {
			m.FreeMemObjects[i] = AddrPair{pair.Start + sizeInBytes, pair.End}
			m.mallocHistory[pair.Start] = sizeInBytes
			return pair.Start, nil
		}
	}
	return 0, errors.New("insufficient space")
}

func NewVmem(memSize int, pageByteSize int) *Vmem {
	m := new(Vmem)
	m.Stack = make([]byte, memSize)
	m.AccessMap = make(map[int]byte)
	m.PAGE_BYTESIZE = pageByteSize
	m.FreeMemObjects = make([]AddrPair, 1)
	m.FreeMemObjects[0] = AddrPair{0, memSize - 1}
	m.mallocHistory = make(map[int]int)
	m.arDisabled = false
	m.faultListeners = make([]FaultListener, 0)
	return m
}

func (m *Vmem) Read(addr int) (byte, error) {
	//If access rights disabled, just read no matter what
	if m.arDisabled {
		return m.Stack[addr], nil
	}
	access := m.AccessMap[m.GetPageAddr(addr)]
	switch access {
	case NO_ACCESS:
		//notify all listeners
		for _, l := range m.faultListeners {
			l(addr, 0)
		}
		return 127, errors.New("access denied")
	case READ_ONLY:
		return m.Stack[addr], nil
	case READ_WRITE:
		return m.Stack[addr], nil
	}
	return 127, errors.New("unknown error")
}

func (m *Vmem) ReadBytes(addr, length int) ([]byte, error) {
	start := addr
	end := addr + length - 1
	if m.arDisabled {
		return m.Stack[start : end+1], nil
	}
	for i := start; i < end; i += m.PAGE_BYTESIZE {
		access := m.AccessMap[m.GetPageAddr(i)]
		switch access {
		case NO_ACCESS:
			//notify all listeners
			for _, l := range m.faultListeners {
				l(addr, 0)
			}
			return nil, errors.New("access denied at location: " + strconv.Itoa(i))
		case READ_WRITE, READ_ONLY:
			continue
		default:
			return nil, errors.New("unknown access value at: " + strconv.Itoa(i))
		}
	}
	return m.Stack[start : end+1], nil
}

func (m *Vmem) Write(addr int, val byte) error {
	if m.arDisabled {
		m.Stack[addr] = val
		return nil
	}
	access := m.AccessMap[m.GetPageAddr(addr)]
	switch access {
	case NO_ACCESS:
		//notify all listeners
		for _, l := range m.faultListeners {
			l(addr, 1)
		}
		return errors.New("access denied")
	case READ_ONLY:
		//notify all listeners
		for _, l := range m.faultListeners {
			l(addr, 1)
		}
		return errors.New("access denied")
	case READ_WRITE:
		m.Stack[addr] = val
		return nil
	}
	return errors.New("unknown error")

}

func (m *Vmem) GetRights(addr int) byte {
	return m.AccessMap[m.GetPageAddr(addr)]
}

func (m *Vmem) SetRights(addr int, access byte) {
	m.AccessMap[m.GetPageAddr(addr)] = access
}

func (m *Vmem) GetPageAddr(addr int) int {
	return addr - addr%m.PAGE_BYTESIZE
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
