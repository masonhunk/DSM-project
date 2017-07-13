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
	fmt.Println("hello")
	assert.Nil(t, err)
	fmt.Println(ptr)
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

func TestREADWRITE(t *testing.T) {

}
