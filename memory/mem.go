package memory

import (
  "errors"
  "strconv"
)

var NO_ACCESS byte = 0
var READ_ONLY byte = 1
var READ_WRITE byte = 2

type VirtualMemory interface {
  Read(addr int) (byte, error)
  ReadBytes(addr, length int) ([]byte, error)
  Write(addr int, val byte) error
  GetRights(addr int) byte
  SetRights(addr int, access byte)
  GetPageAddr(addr int) int
  Malloc(sizeInBytes int) (int, error)
  Free(offset, sizeInBytes int) error
}

type Vmem struct {
  Stack          []byte
  AccessMap      map[int]byte
  PAGE_BYTESIZE  int
  FreeMemObjects []AddrPair
}

func (m *Vmem) Free(offset, sizeInBytes int) error {
  if offset + sizeInBytes >= len(m.Stack) {
    return errors.New("index out of bounds")
  }
  start := offset
  end := offset + sizeInBytes - 1
  var newlist []AddrPair
  j := -1
  for i, pair := range m.FreeMemObjects {
    if pair.End + 1 < start  {
      newlist = append(newlist, pair)
    } else if end + 1 < pair.Start {
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
    if pair.End- pair.Start+ 1 >= sizeInBytes {
      m.FreeMemObjects[i] = AddrPair{pair.Start + sizeInBytes, pair.End}
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
  return m
}

func (m *Vmem) Read(addr int) (byte, error) {
  access := m.AccessMap[m.GetPageAddr(addr)]
  switch access {
  case NO_ACCESS:
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
  for i := start; i < end; i += m.PAGE_BYTESIZE {
    access := m.AccessMap[m.GetPageAddr(i)]
    switch access {
    case NO_ACCESS:
      return nil, errors.New("access denied at location: " + strconv.Itoa(i))
    case  READ_WRITE, READ_ONLY:
      continue
    default:
      return nil, errors.New("unknown access value at: " + strconv.Itoa(i))
    }
  }
  return m.Stack[start:end + 1], nil
}


func (m *Vmem) Write(addr int, val byte) error {
  access := m.AccessMap[m.GetPageAddr(addr)]
  switch access {
  case NO_ACCESS:
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