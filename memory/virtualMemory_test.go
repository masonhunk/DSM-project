package memory

import (
  "testing"
	"github.com/stretchr/testify/assert"
)

func TestNoAccess(t *testing.T) {
  mem := NewVmem(4096, 128)

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
  mem := NewVmem(4096, 128)
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
  mem := NewVmem(4096, 128)
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
	mem := NewVmem(4096, 128)
	mem.Malloc(1024)
	assert.Equal(t, AddrPair{1024, 4095}, mem.FreeMemObjects[0])
	err := mem.Free(0)
	if err != nil {
		t.Error("Expected nil error, got:", err)
	}
	assert.Equal(t, 1, len(mem.FreeMemObjects))
	assert.Equal(t, AddrPair{0, 4095}, mem.FreeMemObjects[0])

	mem = NewVmem(4096, 128)

	mem.Malloc(1024)
	addr1, _ := mem.Malloc(512)
	addr2, _ := mem.Malloc(1024)
	assert.Equal(t, 1024, addr1)
	assert.Equal(t, 1024 + 512, addr2)

	assert.NoError(t, mem.Free(addr1))
	assert.Equal(t, AddrPair{1024, 1024+512-1}, mem.FreeMemObjects[0])
	assert.Equal(t, AddrPair{addr2+1024, 4095}, mem.FreeMemObjects[1])
	mem.Free(addr2)
	assert.Equal(t, AddrPair{1024, 4095}, mem.FreeMemObjects[0])
	mem.Malloc(600)
	assert.Equal(t, AddrPair{1024+600, 4095}, mem.FreeMemObjects[0])
}

/*func TestGoPointers(t *testing.T) {
	arr := make([]byte, 1024)
	fmt.Println(&arr[0])
	fmt.Println(&arr[1])
	arr[0] = 2
	var s *byte
	s = &arr[0]
	*s = 42
	fmt.Println(*&arr[0])
}
*/

