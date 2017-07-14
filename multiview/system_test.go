package multiview

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestInitialize(t *testing.T) {
	mw := NewMultiView()
	mw.Initialize(4096, 128)
	ptr, err := mw.Malloc(1000)
	assert.Nil(t, err)
	fmt.Println("hello")
	mw.Write(ptr, byte(9))
	val, err := mw.Read(ptr)
	fmt.Println(val, err)
	mw.Write(ptr, byte(8))
	val, err = mw.Read(ptr)
	fmt.Println(val, err)
	mw.Shutdown()
}

func TestMalloc(t *testing.T) {
	mw := NewMultiView()
	mw.Initialize(4096, 128)
	ptr, err := mw.Malloc(100)
	_, err2 := mw.Malloc(200)
	assert.Nil(t, err)
	assert.Nil(t, err2)
	err = mw.Free(ptr, 100)
	assert.Nil(t, err)
	ptr3, err3 := mw.Malloc(100)
	assert.Nil(t, err3)
	assert.Equal(t,ptr, ptr3)
	mw.Shutdown()
}

func TestMultipleHosts(t *testing.T) {
	mw1 := NewMultiView()
	mw2 := NewMultiView()
	mw3 := NewMultiView()
	mw4 := NewMultiView()
	mw5 := NewMultiView()

	mw1.Initialize(1024, 32)
	mw2.Join(1024, 32)
	mw3.Join(1024, 32)
	mw4.Join(1024, 32)
	mw5.Join(1024, 32)

	ptr, err := mw2.Malloc(512)
	fmt.Println(ptr, err)

	mw2.Write(ptr, 90)
	res, err := mw3.Read(ptr)
	assert.Equal(t, 90, res)
}
