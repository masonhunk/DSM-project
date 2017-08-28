package treadmarks

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
	"time"
)

func TestNewTreadmarksApi_ReadWriteSingleProc(t *testing.T) {
	tm, err := NewTreadmarksApi(1024, 128, 1, 1, 1)
	assert.Nil(t, err)
	err = tm.Initialize(1000)
	assert.Nil(t, err)
	err = tm.Write(1, byte(2))

	assert.Nil(t, err)
	data, err := tm.Read(1)
	assert.Nil(t, err)
	assert.Equal(t, byte(2), data)
	data, err = tm.Read(200)
	assert.Nil(t, err)
	assert.Equal(t, byte(0), data)
	fmt.Println("boom")
	tm.Shutdown()
}

func TestNewTreadmarksApi_LockAcquireReleaseSingleProc(t *testing.T) {
	tm, err := NewTreadmarksApi(1024, 128, 1, 1, 1)
	assert.Nil(t, err)
	err = tm.Initialize(1000)
	assert.True(t, tm.locks[0].haveToken)
	assert.False(t, tm.locks[0].locked)
	tm.AcquireLock(0)
	assert.True(t, tm.locks[0].haveToken)
	assert.True(t, tm.locks[0].locked)
	tm.ReleaseLock(0)
	assert.True(t, tm.locks[0].haveToken)
	assert.False(t, tm.locks[0].locked)
	tm.Shutdown()
}

func TestNewTreadmarksApi_LockAcquireReleaseMultipleProc_sequential(t *testing.T) {
	tm1, err := NewTreadmarksApi(1024, 128, 2, 1, 1)
	assert.Nil(t, err)
	err = tm1.Initialize(1000)
	assert.Nil(t, err)
	tm2, err := NewTreadmarksApi(1024, 128, 2, 1, 1)
	assert.Nil(t, err)
	err = tm2.Initialize(1001)
	assert.Nil(t, err)
	err = tm2.Join("localhost", 1000)
	assert.Equal(t, uint8(1), tm2.myId)
	assert.Nil(t, err)
	assert.True(t, tm1.locks[0].haveToken)
	assert.False(t, tm1.locks[0].locked)
	assert.False(t, tm2.locks[0].haveToken)
	assert.False(t, tm2.locks[0].locked)
	fmt.Println("First acquire")
	tm1.AcquireLock(0)
	fmt.Println("First acquire passed")
	assert.True(t, tm1.locks[0].haveToken)
	assert.True(t, tm1.locks[0].locked)
	assert.False(t, tm2.locks[0].haveToken)
	assert.False(t, tm2.locks[0].locked)
	fmt.Println("First release")
	tm1.ReleaseLock(0)
	fmt.Println("First release passed")
	assert.True(t, tm1.locks[0].haveToken)
	assert.False(t, tm1.locks[0].locked)
	assert.False(t, tm2.locks[0].haveToken)
	assert.False(t, tm2.locks[0].locked)
	fmt.Println("Second acquire")
	tm2.AcquireLock(0)
	fmt.Println("Second acquire passed")
	assert.False(t, tm1.locks[0].haveToken)
	assert.False(t, tm1.locks[0].locked)
	assert.True(t, tm2.locks[0].haveToken)
	assert.True(t, tm2.locks[0].locked)
	fmt.Println("Second release")
	tm2.ReleaseLock(0)
	fmt.Println("Second release passed")
	assert.False(t, tm1.locks[0].haveToken)
	assert.False(t, tm1.locks[0].locked)
	assert.True(t, tm2.locks[0].haveToken)
	assert.False(t, tm2.locks[0].locked)
	fmt.Println("Third acquire")
	tm1.AcquireLock(0)
	fmt.Println("Third acquire passed")
	assert.True(t, tm1.locks[0].haveToken)
	assert.True(t, tm1.locks[0].locked)
	assert.False(t, tm2.locks[0].haveToken)
	assert.False(t, tm2.locks[0].locked)
	fmt.Println("Third release")
	tm1.ReleaseLock(0)
	fmt.Println("Third release passed")
	assert.True(t, tm1.locks[0].haveToken)
	assert.False(t, tm1.locks[0].locked)
	assert.False(t, tm2.locks[0].haveToken)
	assert.False(t, tm2.locks[0].locked)
}

func TestNewTreadmarksApi_BarrierMultipleProc_sequential(t *testing.T) {
	tm1, err := NewTreadmarksApi(1024, 128, 2, 1, 1)
	assert.Nil(t, err)
	err = tm1.Initialize(1000)
	assert.Nil(t, err)
	tm2, err := NewTreadmarksApi(1024, 128, 2, 1, 1)
	assert.Nil(t, err)
	err = tm2.Initialize(1001)
	assert.Nil(t, err)
	err = tm2.Join("localhost", 1000)
	assert.Equal(t, uint8(1), tm2.myId)
	assert.Nil(t, err)
	done := false
	go func(){
		tm1.Barrier(0)
		done = true
	}()
	assert.False(t, done)
	tm2.Barrier(0)
	time.Sleep(time.Millisecond*100)
	assert.True(t, done)
}

func TestNewTreadmarksApi_LockAcquireReleaseMultipleProc_Concurrent(t *testing.T) {
	tm0, _ := NewTreadmarksApi(1024, 128, 3, 3, 1)
	tm0.Initialize(1000)
	tm1, _ := NewTreadmarksApi(1024, 128, 3, 3, 1)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	tm2, _ := NewTreadmarksApi(1024, 128, 3, 3, 1)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)


	go0, go1, go2 := make(chan bool), make(chan bool), make(chan bool)
	done := make([]int, 0, 100)
	go func(){
		tm := tm0
		g := go0
		<- g
		fmt.Println(tm.myId, " beginning to acquire lock.")
		tm.AcquireLock(1)
		fmt.Println(tm.myId, " Lock acquired")
		done = append(done, 0)
		g <- true
		<- g
		fmt.Println(tm.myId, " beginning to release lock.")
		tm.ReleaseLock(1)
		fmt.Println(tm.myId, " Lock released")
		done = append(done, 0)
	}()
	go func(){
		tm := tm1
		g := go1
		<- g
		fmt.Println(tm.myId, " beginning to acquire lock.")
		tm.AcquireLock(1)
		fmt.Println(tm.myId, " Lock acquired")
		done = append(done, 1)
		g <- true
		<- g
		fmt.Println(tm.myId, " beginning to release lock.")
		tm.ReleaseLock(1)
		fmt.Println(tm.myId, " Lock released")
		done = append(done, 1)
	}()
	go func(){
		tm := tm2
		g := go2
		<- g
		fmt.Println(tm.myId, " beginning to acquire lock.")
		tm.AcquireLock(1)
		fmt.Println(tm.myId, " Lock acquired")
		done = append(done, 2)
		g <- true
		<- g
		fmt.Println(tm.myId, " beginning to release lock.")
		tm.ReleaseLock(1)
		fmt.Println(tm.myId, " Lock released")
		done = append(done, 2)
	}()
	go0 <- true
	time.Sleep(time.Millisecond*10)
	go1 <- true
	time.Sleep(time.Millisecond*10)
	go2 <- true
	time.Sleep(time.Millisecond*10)
	<-go0
	assert.Len(t, done, 1)
	assert.Equal(t, 0, done[0])

	go0 <- true
	<-go1
	time.Sleep(time.Millisecond*10)
	assert.Len(t, done, 3)
	assert.Equal(t, []int{0,0,1}, done)

	go1 <- true
	<-go2
	time.Sleep(time.Millisecond*10)
	assert.Len(t, done, 5)
	assert.Equal(t, []int{0,0,1,1,2}, done)
}

func TestTreadmarksApi_Barrier_MultiplePeers(t *testing.T) {
	tm1, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm1.Initialize(1000)
	tm2, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm2.Initialize(1001)
	tm2.Join("localhost", 1000)
	tm3, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm3.Initialize(1002)
	tm3.Join("localhost", 1000)

	go1, go2, go3 := make(chan bool), make(chan bool), make(chan bool)

	go func(){
		<- go1
		fmt.Println("0 going to barrier")
		tm1.Barrier(1)
		fmt.Println("0 Done")
		go1 <- true
	}()
	go func(){
		<- go2
		fmt.Println("1 going to barrier")
		tm2.Barrier(1)
		fmt.Println("1 Done")
		go2 <- true
	}()
	go func(){
		<- go3
		fmt.Println("2 going to barrier")
		tm3.Barrier(1)
		fmt.Println("2 Done")
		go3 <- true
	}()
	go1 <- true
	go2 <- true
	timeout := false
	select {
	case <-go1:
		timeout = false
	case <- go2:
		timeout = false
	case <- go3:
		timeout = false
	case <-time.After(time.Second * 1):
		timeout = true
	}

	assert.True(t, timeout)
	go3 <- true

	assert.True(t, <- go1)
	assert.True(t, <- go2)

	assert.True(t, <- go3)
}

func TestNewTreadmarksApi_ReadAndWriteSingle(t *testing.T) {
	tm1, _ := NewTreadmarksApi(1024, 128, 1, 3, 3)
	tm1.Initialize(1000)

	go1 := make(chan bool)
	go func() {
		data, err := tm1.Read(1)
		assert.Equal(t, data, byte(0))
		assert.Nil(t, err)
		err = tm1.Write(1, byte(2))
		assert.Nil(t, err)
		data, err = tm1.Read(1)
		assert.Equal(t, data, byte(2))
		assert.Nil(t, err)
		go1 <- true
	}()

	assert.True(t, <- go1)
}

func TestNewTreadmarksApi_ReadAndWriteSingleOther(t *testing.T) {
	tm1, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm1.Initialize(1000)
	tm2, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm2.Initialize(1001)
	tm2.Join("localhost", 1000)

	go2 := make(chan bool)
	go func() {
		data, err := tm2.Read(1)
		assert.Equal(t, data, byte(0))
		assert.Nil(t, err)
		err = tm2.Write(1, byte(2))
		assert.Nil(t, err)
		data, err = tm2.Read(1)
		assert.Equal(t, data, byte(2))
		assert.Nil(t, err)
		go2 <- true
	}()
	assert.True(t, <- go2)
}

func TestNewTreadmarksApi_ReadAndWriteMultiple(t *testing.T) {
	tm0, _ := NewTreadmarksApi(1024, 128, 2, 3, 3)
	tm0.Initialize(1000)
	tm1, _ := NewTreadmarksApi(1024, 128, 2, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)

	go1, go2 := make(chan bool), make(chan bool)
	go func() {
		data, err := tm0.Read(1)
		assert.Equal(t, data, byte(0))
		assert.Nil(t, err)
		err = tm0.Write(1, byte(1))
		assert.Nil(t, err)
		data, err = tm0.Read(1)
		assert.Equal(t, data, byte(1))
		assert.Nil(t, err)
		fmt.Println("0 hit barrier")
		tm0.Barrier(1)
		fmt.Println("0 passed barrier")
		go1 <- true
	}()
	go func() {
		fmt.Println("1 hit barrier")
		tm1.Barrier(1)
		fmt.Println("1 passed barrier")
		data, err := tm1.Read(1)
		assert.Equal(t, data, byte(1))
		assert.Nil(t, err)
		go2 <- true
	}()
	assert.True(t, <- go1)
	assert.True(t, <- go2)
}

func TestNewTreadmarksApi_ReadAndWriteMultiple2(t *testing.T) {
	tm0, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm0.Initialize(1000)
	tm1, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	tm2, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)

	done := make(chan bool, 2)

	go func(){
		tm := tm1
		tm.Write(1, byte(2))
		time.Sleep(time.Millisecond*50)
		tm.AcquireLock(1)
		tm.ReleaseLock(1)
		b, _ := tm.Read(1)
		assert.Equal(t, byte(2), b)
		time.Sleep(time.Millisecond*100)
		tm.AcquireLock(1)
		tm.ReleaseLock(1)
		b, _ = tm.Read(1)
		assert.Equal(t, byte(2), b)
		done <- true
	}()
	go func(){
		tm := tm2
		tm.Write(1, byte(4))
		time.Sleep(time.Millisecond*100)
		tm.AcquireLock(1)
		tm.ReleaseLock(1)
		b, _ :=  tm.Read(1)
		assert.Equal(t, byte(2),b)
		done <- true
	}()
	<- done
	<- done
}

