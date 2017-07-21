package multiview

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	//log.SetOutput(ioutil.Discard)
	mw := NewMultiView()
	mw.Initialize(4096, 128)
	ptr, err := mw.Malloc(1000)
	assert.Nil(t, err)
	mw.Write(ptr, byte(9))
	mw.Read(ptr)
	mw.Write(ptr, byte(8))
	_, err = mw.Read(ptr)
	mw.Shutdown()
}

func TestMalloc(t *testing.T) {
	log.SetOutput(ioutil.Discard)
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
	assert.Equal(t, ptr, ptr3)
	mw.Shutdown()
}

func TestMultipleHosts(t *testing.T) {
	log.SetOutput(ioutil.Discard)
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

	ptr, _ := mw2.Malloc(512)

	mw2.Write(ptr, byte(90))
	res, _ := mw3.Read(ptr)
	assert.Equal(t, byte(90), res)
	mw3.Write(ptr+1, byte(91))

	res1, _ := mw4.Read(ptr + 1)
	res2, _ := mw2.Read(ptr + 1)
	res3, _ := mw5.Read(ptr + 1)
	res4, _ := mw1.Read(ptr + 1)

	assert.Equal(t, byte(91), res1)
	assert.Equal(t, byte(91), res2)
	assert.Equal(t, byte(91), res3)
	assert.Equal(t, byte(91), res4)

	time.Sleep(time.Millisecond * 100)
	mw2.Leave()
	mw3.Leave()
	mw4.Leave()
	mw5.Leave()
	mw1.Shutdown()
}
