package memory

type Page struct {
	AccessRight int
	memory []byte
}

const PAGE_BYTESIZE  = 1024

// NO_ACCESS is nil
const READ_ONLY byte = 1
const READ_WRITE byte = 2


type VirtualMemory interface {
	Read(addr int) (byte, error)
	Write(addr int, val byte) error
	GetRights(addr int) byte
	SetRights(addr int, access byte)
}

type VMem struct {
	stack []byte
	AccessMap map[int]byte
}


func (m *VMem) Read(addr int) (byte, error) {
	access := m.AccessMap[getPageAddr(addr)]
	switch access {
	case nil:
		return nil, error("access denied")
	case READ_ONLY:
		return m.stack[addr], nil
	case READ_WRITE:
		return m.stack[addr], nil
	}
	return nil, error("unknown error")
}

func (m *VMem) Write(addr int, val byte) error {
	access := m.AccessMap[getPageAddr(addr)]
	switch access {
	case nil:
		return error("access denied")
	case READ_ONLY:
		return error("access denied")
	case READ_WRITE:
		m.stack[addr] = val
		return nil
	}
	return error("unknown error")

}

func (m *VMem) GetRights(addr int) byte {
	return m.AccessMap[getPageAddr(addr)]
}

func (m *VMem) SetRights(addr int, access byte) {
	m.AccessMap[getPageAddr(addr)] = access
}


func getPageAddr(addr int) int {
	return addr - addr % PAGE_BYTESIZE
}