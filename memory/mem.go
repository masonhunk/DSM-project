package memory

import (
  "errors"
  "math"
)

const NO_ACCESS byte = 0
const READ_ONLY byte = 1
const READ_WRITE byte = 2

type VirtualMemory interface {
  Read(addr int) (byte, error)
  Write(addr int, val byte) error
  GetRights(addr int) byte
  SetRights(addr int, access byte)
  GetPageAddr(addr int) int
  Malloc(sizeInBytes int) (int, error)
  Free(offset, sizeInBytes int) error
}

type Vmem struct {
  Stack         []byte
  AccessMap     map[int]byte
  PAGE_BYTESIZE int
  freeMemObjects []addrPair
}

func (m *Vmem) Free(offset, sizeInBytes int) error {
  if offset + sizeInBytes >= len(m.Stack) {
    return errors.New("index out of bounds")
  }
  foundOverlap := false
  var newlist []addrPair
  for _, pair := range m.freeMemObjects {
    if pair.end < offset || pair.start > offset + sizeInBytes{
      newlist = append(newlist, pair)
    } else if pair.start <= offset && pair.end >= offset + sizeInBytes {
      return nil
    } else if pair.start <= offset && offset + sizeInBytes >= pair.end {
      newlist = append(newlist, addrPair{min(pair.start, offset),max(pair.end, offset + sizeInBytes)})
      foundOverlap = true
    } else if foundOverlap && pair.start <= offset && pair.end >= offset + sizeInBytes {
      newlist[len(newlist) - 1] = addrPair{newlist[len(newlist) - 1].start, pair.end}
    }
  }
  m.freeMemObjects = newlist
  return nil
}

type addrPair struct {
  start, end int
}

func (m *Vmem) Malloc(sizeInBytes int) (int, error) {
  for i, pair := range m.freeMemObjects{
    if pair.end - pair.start + 1 >= sizeInBytes {
      m.freeMemObjects[i] = addrPair{pair.start + sizeInBytes, pair.end}
      return pair.start, nil
    }
  }
  return 0, errors.New("insufficient space")
}


func NewVmem(memSize int, pageByteSize int) *Vmem {
  m := new(Vmem)
  m.Stack = make([]byte, memSize)
  m.AccessMap = make(map[int]byte)
  m.PAGE_BYTESIZE = pageByteSize
  m.freeMemObjects = make([]addrPair, 1)
  m.freeMemObjects[0] = addrPair{0, memSize - 1}
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