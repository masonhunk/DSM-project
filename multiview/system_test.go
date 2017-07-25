package multiview

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"sync"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	//log.SetOutput(ioutil.Discard)
	mw := NewMultiView()
	mw.Initialize(4096, 128, 1)
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
	mw.Initialize(4096, 128, 1)
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

	mw1.Initialize(1024, 32, 5)
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

func TestMultiview_Lock(t *testing.T) {
	mw1 := NewMultiView()
	mw2 := NewMultiView()
	mw3 := NewMultiView()

	mw1.Initialize(1024, 32, 3)
	mw2.Join(1024, 32)
	mw3.Join(1024, 32)
	group := sync.WaitGroup{}
	group.Add(3)
	started := make(chan bool)
	go func() {
		mw1.Lock(1)
		mw1.Write(100, byte(10))
		mw1.Write(101, byte(11))
		started <- true
		mw1.Release(1)
		group.Done()
	}()
	<-started
	go func() {
		mw2.Lock(1)
		mw2.Write(100, byte(12))
		mw2.Write(101, byte(13))
		mw2.Release(1)
		group.Done()
	}()
	go func() {
		mw3.Lock(1)
		mw3.Write(102, byte(14))
		mw3.Write(103, byte(15))
		mw3.Release(1)
		group.Done()
	}()
	group.Wait()
	res11, _ := mw1.Read(100)
	res12, _ := mw1.Read(101)
	res21, _ := mw2.Read(100)
	res22, _ := mw2.Read(101)
	res31, _ := mw3.Read(100)
	res32, _ := mw3.Read(101)
	assert.Equal(t, byte(12), res11)
	assert.Equal(t, byte(12), res21)
	assert.Equal(t, byte(12), res31)
	assert.Equal(t, byte(13), res12)
	assert.Equal(t, byte(13), res22)
	assert.Equal(t, byte(13), res32)

}
