package tests

import (
  "testing"
  "DSM-project/memory"
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
  
}
