package multiview

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestInitialize(t *testing.T) {
	Initialize(4096, 128)
	ptr, err := mem.Malloc(1000)
	fmt.Println("hello")
	assert.Nil(t, err)
	fmt.Println(ptr)
	mem.Write(ptr, byte(9))
	val, err := mem.Read(ptr)
	fmt.Println(val, err)
	mem.Write(ptr, byte(8))
	val, err = mem.Read(ptr)
	fmt.Println(val, err)

	Shutdown()
}

func TestMalloc(t *testing.T) {
	Initialize(4096, 128)
	_, err := mem.Malloc(1000)
	_, err2 := mem.Malloc(2000)
	assert.Nil(t, err)
	assert.Nil(t, err2)

	Shutdown()
}
