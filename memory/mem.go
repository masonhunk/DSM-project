package memory

import (
	"errors"
	"sync"
	"fmt"
)

var AccessDeniedErr = errors.New("access denied")

const NO_ACCESS byte = 0
const READ_ONLY byte = 1
const READ_WRITE byte = 2

type FaultListener func(addr int, length int, faultType byte, accessType string, value []byte) error //faultType = 0 => read fault, = 1 => write fault

type VirtualMemory interface {
	Read(addr int) (byte, error)
	ReadBytes(addr, length int) ([]byte, error)
	Write(addr int, val byte) error
	WriteBytes(addr int, val []byte) error
	PrivilegedRead(addr, length int) []byte
	PrivilegedWrite(addr int, data []byte) error
	GetRights(addr int) byte
	GetRightsList(addr []int) []byte
	SetRights(addr int, access byte)
	SetRightsList(addr []int, access byte)
	GetPageAddr(addr int) int
	Malloc(sizeInBytes int) (int, error)
	Free(pointer int) error
	GetPageSize() int
	Size() int
	AccessRightsDisabled(b bool)
	AddFaultListener(l FaultListener)
}

type Vmem struct {
	Stack          []byte
	AccessMap      map[int]byte
	PAGE_BYTESIZE  int
	FreeMemObjects []AddrPair
	mallocHistory  map[int]int //maps address to length
	arDisabled     bool
	faultListeners []FaultListener
	accessLock		*sync.Mutex

	mutex          *sync.Mutex
}

func (m *Vmem) PrivilegedRead(addr, length int) []byte {
	read := make([]byte, length)
	copy(read, m.Stack[addr:addr+length])
	return read
}

func (m *Vmem) PrivilegedWrite(addr int, data []byte) error {
	if addr+len(data) > len(m.Stack) {
		fmt.Println(len(m.Stack))
		fmt.Println(len(m.Stack[addr: addr+len(data)]))
		fmt.Println(len(data))
	}
	copy(m.Stack[addr: addr+len(data)], data)
	return nil
}

func (m *Vmem) AddFaultListener(l FaultListener) {
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
	m.Stack = make([]byte, max(memSize, pageByteSize))
	m.AccessMap = make(map[int]byte)
	m.PAGE_BYTESIZE = pageByteSize
	m.FreeMemObjects = make([]AddrPair, 1)
	m.FreeMemObjects[0] = AddrPair{0, max(memSize, pageByteSize) - 1}
	m.mallocHistory = make(map[int]int)
	m.arDisabled = false
	m.faultListeners = make([]FaultListener, 0)
	m.mutex = new(sync.Mutex)
	m.accessLock = new(sync.Mutex)
	return m
}

func (m *Vmem) Read(addr int) (byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	//If access rights disabled, just read no matter what
	if m.arDisabled {
		return m.Stack[addr], nil
	}
	access := m.GetRights(addr)

	if access == NO_ACCESS {
		for _, l := range m.faultListeners {
			err := l(addr,1,  0, "READ", []byte{0})
			if err == nil {
				return m.Stack[addr], nil
			}
		}
		return m.Stack[addr], errors.New("access denied")
	}
	return m.Stack[addr], nil
}

func (m *Vmem) ReadBytes(addr, length int) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	firstPageAddr := m.GetPageAddr(addr)
	lastPageAddr := m.GetPageAddr(addr+ length)
	result := make([]byte, length)

	addrList := make([]int, 0)
	for tempAddr := firstPageAddr; tempAddr <= lastPageAddr; tempAddr = tempAddr + m.GetPageSize() {
		addrList = append(addrList, tempAddr)
	}
	access := m.GetRightsList(addrList)
Loop:
	for i := range access {
		if access[i] == NO_ACCESS {
			for _, l := range m.faultListeners {
				err := l(addr, length,1, "READ", nil)
				if err == nil {
					break Loop
				}
			}
			return nil, errors.New("access denied")
		}
	}
	copy(result, m.Stack[addr:addr+length])
	return result, nil
}

func (m *Vmem) Write(addr int, val byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.arDisabled {
		m.Stack[addr] = val
		return nil
	}
	access := m.GetRights(m.GetPageAddr(addr))
	if access != READ_WRITE {
		for _, l := range m.faultListeners {
			err := l(addr, 1, 1, "WRITE", []byte{val})
			if err == nil {
				m.Stack[addr] = val
				return nil
			}
		}
		return errors.New("access denied")
	} else {
		m.Stack[addr] = val
		return nil
	}

}

func (m *Vmem) WriteBytes(addr int, val []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	length := len(val)
	firstPageAddr := m.GetPageAddr(addr)
	lastPageAddr := m.GetPageAddr(addr + length-1)
	addrList := make([]int, 0)
	for tempAddr := firstPageAddr; tempAddr <= lastPageAddr; tempAddr = tempAddr + m.GetPageSize() {
		addrList = append(addrList, tempAddr)
	}
	access := m.GetRightsList(addrList)
Loop:
	for i := range access {
		if access[i] != READ_WRITE {
			for _, l := range m.faultListeners {
				err := l(addrList[i], length, 1, "WRITE", val)
				if err == nil {
					break Loop
				}
			}
			return errors.New("access denied")
		}
	}
	copy(m.Stack[addr:addr+length], val)
	return nil
}

func (m *Vmem) GetRights(addr int) byte {
	m.accessLock.Lock()
	res := m.AccessMap[m.GetPageAddr(addr)]
	m.accessLock.Unlock()
	return res
}

func (m *Vmem) GetRightsList(addrList []int) []byte{
	m.accessLock.Lock()
	defer m.accessLock.Unlock()
	result := make([]byte, len(addrList))
	for i, addr := range addrList{
		result[i] = m.AccessMap[m.GetPageAddr(addr)]
	}
	return result
}

func (m *Vmem) SetRights(addr int, access byte) {
	m.accessLock.Lock()
	m.AccessMap[m.GetPageAddr(addr)] = access
	m.accessLock.Unlock()
}

func (m *Vmem) SetRightsList(addrList []int, access byte) {
	m.accessLock.Lock()
	defer m.accessLock.Unlock()
	for _, addr := range addrList {
		m.AccessMap[m.GetPageAddr(addr)] = access
	}
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
