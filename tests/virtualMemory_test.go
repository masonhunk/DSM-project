package tests

import (
  "testing"
  "DSM-project/memory"
	"github.com/stretchr/testify/assert"

)

func TestNoAccess(t *testing.T) {
  mem := memory.NewVmem(4096, 128)

  if r := mem.GetRights(125); r != 0 {
	t.Error("Expected NO_ACCESS rights (= 0), got:", r)
  }
  if _, err := mem.Read(125); err == nil {
	t.Error("Expected no access (nil)")
  }
  if err := mem.Write(125, 90); err == nil {
	t.Error("Expected no access (nil)")
  }

}

func TestCorrectPageAddr(t *testing.T) {
  mem := memory.NewVmem(4096, 128)
  if addr := mem.GetPageAddr(57); addr != 0 {
	t.Error("Expected address 0, got:", addr)
  }
  if addr := mem.GetPageAddr(1029); addr != 1024 {
	t.Error("Expected address 1024, got:", addr)
  }
  if addr := mem.GetPageAddr(5051); addr != 128*39 {
	t.Error("Expected address 4992, got:", addr)
  }
}

func TestMalloc(t *testing.T) {
  mem := memory.NewVmem(4096, 128)
  addr1, err1 := mem.Malloc(512)
  addr2, err2 := mem.Malloc(1024)
	if addr1 != 0 || err1 != nil {
		t.Error("Expected address 0, got:", addr1)
		t.Error("Expected nil error, got:", err1)
	}
	if addr2 != 512 {
		t.Error("Expected address 512, got:", addr2)
		t.Error("Expected nil error, got:", err2)
	}
}

func TestFreeMemory(t *testing.T) {
	mem := memory.NewVmem(4096, 128)
	mem.Malloc(1024)
	err := mem.Free(0, 512)
	if err != nil {
		t.Error("Expected nil error, got:", err)
	}
	assert.Equal(t, 2, len(mem.FreeMemObjects))
	assert.Equal(t, memory.AddrPair{0, 511}, mem.FreeMemObjects[0])
	assert.Equal(t, memory.AddrPair{1024, 4095}, mem.FreeMemObjects[1])
}
