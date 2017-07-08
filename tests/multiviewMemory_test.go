package tests

import (
	"testing"
	"DSM-project/memory"
	"github.com/stretchr/testify/assert"
	"DSM-project/multiview"
	"fmt"
)

func TestMultiViewMalloc(t *testing.T) {
	m := memory.NewVmem(4096, 128)
	mem := multiview.NewMVMem(m)

	ptr, err := mem.Malloc(100)
	assert.Nil(t, err)
	assert.Equal(t, 0, ptr)
	vp := mem.AddrToPage[0]
	assert.Equal(t, 100, vp.Length)
	assert.Equal(t, 0, vp.Offset)

	ptr, err = mem.Malloc(300)
	assert.Nil(t, err)
	assert.Equal(t, 228, ptr)
}

func TestCorrectAddressTranslation(t *testing.T) {
	m := memory.NewVmem(4096, 128)
	mem := multiview.NewMVMem(m)

	ptr, _:= mem.Malloc(330)
	ptr1, _ := mem.Malloc(400)

	assert.Equal(t, 0, ptr)
	assert.Equal(t, 330-256 + 384, ptr1)
	mem.Write(460, 42)
	mem.Write(461, 43)
	res, _:= mem.Read(460)
	assert.Equal(t, byte(42), res)
	addr := mem.AddrToPage[m.GetPageAddr(ptr1)/m.GetPageSize()].PageAddr+(460-384)
	//addr = 332 because we first malloc'ed 330, and then wrote to the second entry in the next malloc => addr 332
	assert.Equal(t, byte(42), m.Stack[addr])
	assert.Equal(t, byte(43), m.Stack[addr + 1])
}

func TestHowReferenceTypesWork(t *testing.T) {
	m := memory.NewVmem(4096, 128)
	mem := multiview.NewMVMem(m)
	mem.Malloc(100)
	assert.NoError(t, mem.Write(50, 42))
	assert.Equal(t, byte(42), m.Stack[50])

	res, err := mem.Read(50)
	assert.Nil(t, err)
	assert.Equal(t, byte(42), res)
}

func TestMVMFree(t *testing.T) {
	m := memory.NewVmem(4096, 128)
	mem := multiview.NewMVMem(m)
	mem.Malloc(100)
	fmt.Println(mem.AddrToPage)

	assert.Equal(t, memory.AddrPair{100, 4095},m.FreeMemObjects[0])
	assert.Nil(t,mem.Free(0, 100))
	assert.Equal(t, memory.AddrPair{0, 4095},m.FreeMemObjects[0])
	assert.Len(t, mem.AddrToPage, 0)

	addr1, err := mem.Malloc(460)
	addr2, er2 := mem.Malloc(900)

	assert.Nil(t, err)
	assert.Nil(t, er2)
	fmt.Println(addr1, addr2)
	fmt.Println(mem.AddrToPage)
	mem.Write(addr2 + 400, 42)

	err = mem.Free(addr2, 400)
	assert.Nil(t, err)
	fmt.Println(mem.AddrToPage)
	_, ok := mem.Read(addr2 + 400)
	assert.Nil(t, ok)
	//assert.Equal(t, 42, res)
}

