package multiview

import (
	"fmt"
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

	ptr, _ := mw1.Malloc(512)

	group := sync.WaitGroup{}
	group.Add(3)
	started := make(chan bool)
	go func() {
		mw1.Lock(1)
		mw1.Write(ptr, byte(10))
		mw1.Write(ptr+1, byte(11))
		started <- true
		mw1.Release(1)
		group.Done()
	}()
	<-started
	go func() {
		mw2.Lock(1)
		mw2.Write(ptr, byte(12))
		mw2.Write(ptr+1, byte(13))
		mw2.Release(1)
		group.Done()
	}()
	go func() {
		mw3.Lock(1)
		mw3.Write(ptr+2, byte(14))
		mw3.Write(ptr+3, byte(15))
		mw3.Release(1)
		group.Done()
	}()
	group.Wait()
	res11, _ := mw1.Read(ptr)
	res12, _ := mw1.Read(ptr + 1)
	res21, _ := mw2.Read(ptr)
	res22, _ := mw2.Read(ptr + 1)
	res31, _ := mw3.Read(ptr)
	res32, _ := mw3.Read(ptr + 1)

	assert.Equal(t, byte(12), res11)
	assert.Equal(t, byte(12), res21) //passed
	assert.Equal(t, byte(12), res31)
	assert.Equal(t, byte(13), res12)
	assert.Equal(t, byte(13), res22) //passed
	assert.Equal(t, byte(13), res32)

	mw3.Leave()
	mw2.Leave()
	mw1.Shutdown()
}

func TestMultiview_Barrier(t *testing.T) {
	mw1 := NewMultiView()
	mw2 := NewMultiView()
	mw3 := NewMultiView()

	mw1.Initialize(1024, 32, 3)
	mw2.Join(1024, 32)
	mw3.Join(1024, 32)

	ptr, _ := mw1.Malloc(512)
	group := sync.WaitGroup{}
	group.Add(2)
	started := make(chan bool)
	go func() {
		started <- true
		mw1.Write(ptr, byte(10))
		group.Done()
		mw1.Barrier(1)
		mw1.Write(ptr+1, byte(11))
		group.Done()
	}()
	<-started

	go func() {
		started <- true
		group.Done()
		mw2.Barrier(1)
		mw2.Write(ptr, byte(12))
		group.Done()
	}()
	<-started
	group.Wait()
	group.Add(2)
	//not yet over barrier
	res1, _ := mw1.Read(ptr)
	res2, _ := mw2.Read(ptr)
	res3, _ := mw3.Read(ptr)

	assert.Equal(t, byte(10), res1)
	assert.Equal(t, byte(10), res2)
	assert.Equal(t, byte(10), res3)

	res1, _ = mw1.Read(ptr + 1)
	res2, _ = mw2.Read(ptr + 1)
	res3, _ = mw3.Read(ptr + 1)

	assert.Equal(t, byte(0), res1)
	assert.Equal(t, byte(0), res2)
	assert.Equal(t, byte(0), res3)

	mw3.Barrier(1)
	group.Wait()
	res1, _ = mw1.Read(ptr)
	res2, _ = mw2.Read(ptr)
	res3, _ = mw3.Read(ptr)

	assert.Equal(t, byte(12), res1)
	assert.Equal(t, byte(12), res2)
	assert.Equal(t, byte(12), res3)

	res1, _ = mw1.Read(ptr + 1)
	res2, _ = mw2.Read(ptr + 1)
	res3, _ = mw3.Read(ptr + 1)

	assert.Equal(t, byte(11), res1)
	assert.Equal(t, byte(11), res2)
	assert.Equal(t, byte(11), res3)

	mw2.Leave()
	mw3.Leave()
	mw1.Shutdown()
}

func TestConsecutiveWritesAndReads(t *testing.T) {
	mw1 := NewMultiView()
	mw2 := NewMultiView()

	mw1.Initialize(1024, 32, 2)
	mw2.Join(1024, 32)

	ptr, _ := mw1.Malloc(256)

	mw1.Read(ptr)
	mw2.Read(ptr)

	fmt.Println(mw1.Write(ptr, byte(10)))
	val, _ := mw1.Read(ptr)
	assert.Equal(t, byte(10), val)
	res, _ := mw2.Read(ptr)
	assert.Equal(t, byte(10), res)
	mw1.Write(ptr, byte(12))
	res, _ = mw2.Read(ptr)
	assert.Equal(t, byte(12), res)

	mw2.Leave()
	mw1.Shutdown()
}

func TestReacquireLock(t *testing.T) {
	mw1 := NewMultiView()
	mw2 := NewMultiView()
	mw1.Initialize(1024, 32, 2)
	mw2.Join(1024, 32)
	started := make(chan bool)
	done := make(chan bool)
	mw1.Lock(1)
	go func() {
		started <- true
		mw2.Lock(1)
		done <- true
	}()

	mw1.Lock(1)
	<-started
	mw1.Release(1)
	<-done
	mw2.Release(1)

	time.Sleep(100 * time.Millisecond)
	mw2.Leave()
	mw1.Shutdown()
}

func TestMemoryMalloc(t *testing.T) {
	mw1 := NewMultiView()
	mw1.Initialize(4104, 4096, 1)
	addr, err := mw1.Malloc(4)
	assert.Nil(t, err)
	assert.Equal(t, 2*4096, addr)

	//try alloc'ing more memory than available
	mw1 = NewMultiView()
	mw1.Initialize(2*4096, 4096, 1)
	addr, err = mw1.Malloc(2 * 4096)
	assert.Nil(t, err)
	assert.Equal(t, 2*4096, addr)
	_, err1 := mw1.Malloc(2)
	assert.NotNil(t, err1)

}
