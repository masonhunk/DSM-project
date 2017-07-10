package multiview

import (
	"testing"
	"DSM-project/memory"
	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestMultiViewMalloc(t *testing.T) {
	m := memory.NewVmem(4096, 128)
	mem := NewMVMem(m)

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
	mem := NewMVMem(m)

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
	mem := NewMVMem(m)
	mem.Malloc(100)
	assert.NoError(t, mem.Write(50, 42))
	assert.Equal(t, byte(42), m.Stack[50])

	res, err := mem.Read(50)
	assert.Nil(t, err)
	assert.Equal(t, byte(42), res)
}

func TestMVMFree(t *testing.T) {
	m := memory.NewVmem(4096, 128)
	mem := NewMVMem(m)
	mem.Malloc(1000)

	assert.Equal(t, memory.AddrPair{1000, 4095},m.FreeMemObjects[0])
	assert.Nil(t,mem.Free(0))
	assert.Equal(t, memory.AddrPair{0, 4095},m.FreeMemObjects[0])
	assert.Len(t, mem.FreeVPages, 1)
	assert.Equal(t, interval{0, 999}, mem.FreeVPages[0])

	m = memory.NewVmem(4096, 128)
	mem = NewMVMem(m)

	ptr, err := mem.Malloc(1000)
	ptr2, err2 := mem.Malloc(2000)
	assert.Nil(t, err)
	assert.Nil(t, err2)
	assert.Equal(t, 0, ptr)
	assert.Equal(t, 1128, ptr2)

	mem.Write(ptr2, 99)
	assert.Equal(t, byte(99), m.Stack[1000])

	err = mem.Free(ptr)
	assert.Nil(t, err)
	assert.Equal(t, interval{0, 999}, mem.FreeVPages[0])

	adr3, err3 := mem.Malloc(500)
	assert.Nil(t, err3)
	assert.Equal(t, 0, adr3)

	m = memory.NewVmem(4096, 128)
	mem = NewMVMem(m)

	ptr1, err1 := mem.Malloc(1000)
	ptr2, err2 = mem.Malloc(500)
	ptr3, err3 := mem.Malloc(500)
	_, err4 := mem.Malloc(500)

	assert.Nil(t, err1)
	assert.Nil(t, err2)
	assert.Nil(t, err3)
	assert.Nil(t, err4)

	err1 = mem.Free(ptr1)
	err2 = mem.Free(ptr3)

	assert.Nil(t, err1)
	assert.Nil(t, err2)

	assert.Equal(t, interval{0, 999}, mem.FreeVPages[0])
	assert.Equal(t, interval{ptr3, ptr3+500-1}, mem.FreeVPages[1])
	assert.Equal(t, memory.AddrPair{1500, 1999}, m.FreeMemObjects[1])
	mem.Write(1756, 99)
}

func TestAccessRights(t *testing.T) {
	m := memory.NewVmem(4096, 128)
	mem := NewMVMem(m)
	ptr, _ := mem.Malloc(1000)

	assert.Equal(t, byte(0), memory.NO_ACCESS)
	assert.Equal(t, byte(1), memory.READ_ONLY)
	assert.Equal(t, byte(2), memory.READ_WRITE)

	_, err := mem.Read(ptr)
	assert.NoError(t, err)
	assert.NoError(t, mem.Write(ptr, 10))

	mem.SetRights(ptr, memory.NO_ACCESS)
	_, err = mem.Read(ptr)
	assert.NotNil(t, err)
	assert.NotNil(t, mem.Write(ptr, 20))
	mem.SetRights(ptr, memory.READ_ONLY)
	val, err := mem.Read(ptr)
	assert.Equal(t, byte(10), val)
	assert.Nil(t, err)
	assert.NotNil(t, mem.Write(ptr + 10, byte(20)))
}

func TestMVMem_ReadMinipage(t *testing.T) {
	m := memory.NewVmem(4096, 128)
	mem := NewMVMem(m)
	mem.Malloc(1000)

	ptr, _ := mem.Malloc(500)
	err1 := mem.Write(ptr + 5, byte(10))
	err2 := mem.Write(ptr + 6, byte(11))
	mem.Write(ptr + 7, byte(12))
	assert.Nil(t, err1, err2)

	res, err := mem.ReadMinipage(ptr + 4)
	assert.Nil(t, err)
	val, _ := mem.vm.Read(1006)
	assert.Equal(t, byte(11), val)
	val, _ = mem.Read(ptr + 6)
	assert.Equal(t, byte(11), val)
	assert.Equal(t, res[5], byte(10))
	assert.Equal(t, res[6], byte(11))
	assert.Equal(t, res[7], byte(12))
	assert.Equal(t, res[4], byte(0))
	assert.Equal(t, res[8], byte(0))
}

