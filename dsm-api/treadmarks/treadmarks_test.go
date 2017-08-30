package treadmarks

import (
	"DSM-project/dsm-api"
	"DSM-project/utils"
	"fmt"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
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
	tm1.Shutdown()
	tm2.Shutdown()
}

func TestNewTreadmarksApi_BarrierMultipleProc_sequential(t *testing.T) {
	tm1, err := NewTreadmarksApi(1024, 128, 2, 1, 1)
	assert.Nil(t, err)
	err = tm1.Initialize(1000)
	defer tm1.Shutdown()
	assert.Nil(t, err)
	tm2, err := NewTreadmarksApi(1024, 128, 2, 1, 1)
	assert.Nil(t, err)
	err = tm2.Initialize(1001)
	defer tm2.Shutdown()
	assert.Nil(t, err)
	err = tm2.Join("localhost", 1000)
	assert.Equal(t, uint8(1), tm2.myId)
	assert.Nil(t, err)
	done := false
	go func() {
		tm1.Barrier(0)
		done = true
	}()
	assert.False(t, done)
	tm2.Barrier(0)
	time.Sleep(time.Millisecond * 100)
	assert.True(t, done)

}

func TestNewTreadmarksApi_LockAcquireReleaseMultipleProc_Concurrent(t *testing.T) {
	tm0, _ := NewTreadmarksApi(1024, 128, 3, 3, 1)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadmarksApi(1024, 128, 3, 3, 1)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadmarksApi(1024, 128, 3, 3, 1)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	go0, go1, go2 := make(chan bool), make(chan bool), make(chan bool)
	done := make([]int, 0, 100)
	go func() {
		tm := tm0
		g := go0
		<-g
		fmt.Println(tm.myId, " beginning to acquire lock.")
		tm.AcquireLock(1)
		fmt.Println(tm.myId, " Lock acquired")
		done = append(done, 0)
		g <- true
		<-g
		fmt.Println(tm.myId, " beginning to release lock.")
		tm.ReleaseLock(1)
		fmt.Println(tm.myId, " Lock released")
		done = append(done, 0)
	}()
	go func() {
		tm := tm1
		g := go1
		<-g
		fmt.Println(tm.myId, " beginning to acquire lock.")
		tm.AcquireLock(1)
		fmt.Println(tm.myId, " Lock acquired")
		done = append(done, 1)
		g <- true
		<-g
		fmt.Println(tm.myId, " beginning to release lock.")
		tm.ReleaseLock(1)
		fmt.Println(tm.myId, " Lock released")
		done = append(done, 1)
	}()
	go func() {
		tm := tm2
		g := go2
		<-g
		fmt.Println(tm.myId, " beginning to acquire lock.")
		tm.AcquireLock(1)
		fmt.Println(tm.myId, " Lock acquired")
		done = append(done, 2)
		g <- true
		<-g
		fmt.Println(tm.myId, " beginning to release lock.")
		tm.ReleaseLock(1)
		fmt.Println(tm.myId, " Lock released")
		done = append(done, 2)
	}()
	go0 <- true
	time.Sleep(time.Millisecond * 10)
	go1 <- true
	time.Sleep(time.Millisecond * 10)
	go2 <- true
	time.Sleep(time.Millisecond * 10)
	<-go0
	assert.Len(t, done, 1)
	assert.Equal(t, 0, done[0])

	go0 <- true
	<-go1
	time.Sleep(time.Millisecond * 10)
	assert.Len(t, done, 3)
	assert.Equal(t, []int{0, 0, 1}, done)

	go1 <- true
	<-go2
	time.Sleep(time.Millisecond * 10)
	assert.Len(t, done, 5)
	assert.Equal(t, []int{0, 0, 1, 1, 2}, done)
	go2 <- true
}

func TestTreadmarksApi_Barrier_MultiplePeers(t *testing.T) {
	tm1, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm1.Initialize(1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm2.Initialize(1001)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()
	tm3, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm3.Initialize(1002)
	tm3.Join("localhost", 1000)
	defer tm3.Shutdown()

	go1, go2, go3 := make(chan bool), make(chan bool), make(chan bool)

	go func() {
		<-go1
		fmt.Println("0 going to barrier")
		tm1.Barrier(1)
		fmt.Println("0 Done")
		go1 <- true
	}()
	go func() {
		<-go2
		fmt.Println("1 going to barrier")
		tm2.Barrier(1)
		fmt.Println("1 Done")
		go2 <- true
	}()
	go func() {
		<-go3
		fmt.Println("2 going to barrier")
		tm3.Barrier(1)
		fmt.Println("2 Done")
		go3 <- true
	}()
	go1 <- true
	go2 <- true
	assert.True(t, timeout(go1))
	assert.True(t, timeout(go2))
	assert.True(t, timeout(go3))

	go3 <- true

	assert.False(t, timeout(go1))
	assert.False(t, timeout(go2))
	assert.False(t, timeout(go3))
}

func TestNewTreadmarksApi_ReadAndWriteSingle(t *testing.T) {
	tm1, _ := NewTreadmarksApi(1024, 128, 1, 3, 3)
	tm1.Initialize(1000)
	defer tm1.Shutdown()

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

	assert.True(t, <-go1)
}

func TestNewTreadmarksApi_ReadAndWriteSingleOther(t *testing.T) {
	tm1, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm1.Initialize(1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm2.Initialize(1001)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

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
	assert.True(t, <-go2)
}

func TestNewTreadmarksApi_ReadAndWriteMultiple(t *testing.T) {
	tm0, _ := NewTreadmarksApi(1024, 128, 2, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadmarksApi(1024, 128, 2, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()

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
	assert.True(t, <-go1)
	assert.True(t, <-go2)
}

func TestNewTreadmarksApi_ReadAndWriteMultiple2(t *testing.T) {
	tm0, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	done := make(chan bool, 2)

	go func() {
		tm := tm1
		tm.Write(1, byte(2))
		time.Sleep(time.Millisecond * 50)
		tm.AcquireLock(1)
		tm.ReleaseLock(1)
		b, _ := tm.Read(1)
		assert.Equal(t, byte(2), b)
		time.Sleep(time.Millisecond * 100)
		tm.AcquireLock(1)
		tm.ReleaseLock(1)
		fmt.Println("lock released")
		b, _ = tm.Read(1)
		assert.Equal(t, byte(2), b)
		done <- true
	}()
	go func() {
		tm := tm2
		tm.Write(1, byte(4))
		time.Sleep(time.Millisecond * 100)
		tm.AcquireLock(1)
		tm.ReleaseLock(1)
		b, _ := tm.Read(1)
		assert.Equal(t, byte(2), b)
		done <- true
	}()
	<-done
	<-done
}

func TestNewTreadmarksApi_ReadAndWriteMultiple3(t *testing.T) {
	runtime.GOMAXPROCS(2)
	tm0, _ := NewTreadmarksApi(1024*4, 128, 2, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadmarksApi(1024*4, 128, 2, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	done := make(chan bool, 2)
	go func() {
		tm := tm0
		tm.Write(0, 1)
		fmt.Println(tm.myId, " at barrier 0")
		tm.Barrier(0)
		fmt.Println(tm.myId, " past barrier 0")
		for {
			tm.AcquireLock(0)
			n, _ := tm.Read(0)
			if !(n < 200) {
				tm.ReleaseLock(0)
				break
			}
			tm.Write(0, n+10)
			tm.ReleaseLock(0)
			var k uint8
			for k = n; k < n+10 && k < 200; k++ {
				tm.Write(int(k), k)
			}
			fmt.Println(tm.myId, " done iterating n=", k)
		}
		fmt.Println(tm.myId, "at barrier 1")
		tm.Barrier(1)

		done <- true
	}()
	go func() {
		tm := tm1
		fmt.Println(tm.myId, " at barrier 0")
		tm.Barrier(0)
		fmt.Println(tm.myId, " past barrier 0")
		for {
			tm.AcquireLock(0)
			n, _ := tm.Read(0)
			if !(n < 200) {
				tm.ReleaseLock(0)
				break
			}
			tm.Write(0, n+10)
			tm.ReleaseLock(0)
			var k uint8
			for k = n; k < n+10 && k < 200; k++ {
				tm.Write(int(k), k)
			}
			fmt.Println(tm.myId, " done iterating n=", k)
		}
		fmt.Println(tm.myId, "at barrier 1")
		tm.Barrier(1)

		done <- true
	}()
	<-done
	<-done
	var n uint8
	for n = 1; n < 200; n++ {
		val, _ := tm0.Read(int(n))
		assert.Equal(t, n, val)
	}
	for n = 1; n < 200; n++ {
		val, _ := tm1.Read(int(n))
		assert.Equal(t, n, val)
	}
	fmt.Println(tm0.procarray[0])
	fmt.Println(tm1.procarray[0])
	fmt.Println(tm0.procarray[1])
	fmt.Println(tm1.procarray[1])
}

func TestNewTreadmarksApi_locks(t *testing.T) {
	tm0, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadmarksApi(1024, 128, 3, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	//go0:= make(chan bool)
	//go1 := make(chan bool)
	go2 := make(chan bool, 1)

	var lockId uint8 = 0

	lock0, lock1, lock2 := tm0.locks[lockId], tm1.locks[lockId], tm2.locks[lockId]

	assert.False(t, lock0.locked)
	assert.False(t, lock1.locked)
	assert.False(t, lock2.locked)
	assert.True(t, lock0.haveToken)
	assert.False(t, lock1.haveToken)
	assert.False(t, lock2.haveToken)
	assert.Nil(t, lock0.nextTimestamp)
	assert.Nil(t, lock1.nextTimestamp)
	assert.Nil(t, lock2.nextTimestamp)
	assert.Equal(t, tm0.getManagerId(lockId), lock0.last)
	assert.Equal(t, tm1.getManagerId(lockId), lock1.last)
	assert.Equal(t, tm2.getManagerId(lockId), lock2.last)

	tm0.AcquireLock(lockId)

	assert.True(t, lock0.locked)
	assert.False(t, lock1.locked)
	assert.False(t, lock2.locked)
	assert.True(t, lock0.haveToken)
	assert.False(t, lock1.haveToken)
	assert.False(t, lock2.haveToken)
	assert.Nil(t, lock0.nextTimestamp)
	assert.Nil(t, lock1.nextTimestamp)
	assert.Nil(t, lock2.nextTimestamp)
	assert.Equal(t, tm0.getManagerId(lockId), lock0.last)
	assert.Equal(t, tm1.getManagerId(lockId), lock1.last)
	assert.Equal(t, tm2.getManagerId(lockId), lock2.last)

	tm0.ReleaseLock(lockId)
	fmt.Println("Boom")

	assert.False(t, lock0.locked)
	assert.False(t, lock1.locked)
	assert.False(t, lock2.locked)
	assert.True(t, lock0.haveToken)
	assert.False(t, lock1.haveToken)
	assert.False(t, lock2.haveToken)
	assert.Nil(t, lock0.nextTimestamp)
	assert.Nil(t, lock1.nextTimestamp)
	assert.Nil(t, lock2.nextTimestamp)
	assert.Equal(t, tm0.getManagerId(lockId), lock0.last)
	assert.Equal(t, tm1.getManagerId(lockId), lock1.last)
	assert.Equal(t, tm2.getManagerId(lockId), lock2.last)

	tm1.AcquireLock(lockId)
	fmt.Println("Boom")
	assert.False(t, lock0.locked)
	assert.True(t, lock1.locked)
	assert.False(t, lock2.locked)
	assert.False(t, lock0.haveToken)
	assert.True(t, lock1.haveToken)
	assert.False(t, lock2.haveToken)
	assert.Nil(t, lock0.nextTimestamp)
	assert.Nil(t, lock1.nextTimestamp)
	assert.Nil(t, lock2.nextTimestamp)
	assert.Equal(t, uint8(1), lock0.last)
	assert.Equal(t, tm1.myId, lock1.last)
	assert.Equal(t, tm2.getManagerId(lockId), lock2.last)
	fmt.Println("Boom")
	go func() {
		go2 <- true
		fmt.Println("Boom1")
		tm2.AcquireLock(lockId)
		fmt.Println("Boom2")
		go2 <- true

	}()
	<-go2
	time.Sleep(time.Millisecond * 100)
	assert.False(t, lock0.locked)
	assert.True(t, lock1.locked)
	assert.False(t, lock2.locked)
	assert.False(t, lock0.haveToken)
	assert.True(t, lock1.haveToken)
	assert.False(t, lock2.haveToken)
	assert.Nil(t, lock0.nextTimestamp)
	assert.NotNil(t, lock1.nextTimestamp)
	assert.Nil(t, lock2.nextTimestamp)
	assert.Equal(t, uint8(2), lock0.last)
	assert.Equal(t, uint8(2), lock1.nextId)
	assert.Equal(t, tm2.getManagerId(lockId), lock2.nextId)
	fmt.Println("Boom")
	tm1.ReleaseLock(lockId)
	fmt.Println("Boom")
	<-go2
	fmt.Println("Boom")
	time.Sleep(time.Millisecond * 100)
	assert.False(t, lock0.locked)
	assert.False(t, lock1.locked)
	assert.True(t, lock2.locked)
	assert.False(t, lock0.haveToken)
	assert.False(t, lock1.haveToken)
	assert.True(t, lock2.haveToken)
	assert.Nil(t, lock0.nextTimestamp)
	assert.Nil(t, lock1.nextTimestamp)
	assert.Nil(t, lock2.nextTimestamp)
	assert.Equal(t, uint8(2), lock0.last)
	assert.Equal(t, tm1.getManagerId(lockId), lock1.last)
	assert.Equal(t, tm2.myId, lock2.last)
	assert.Equal(t, tm1.getManagerId(lockId), lock1.nextId)
	assert.Equal(t, tm2.getManagerId(lockId), lock2.nextId)
}

func TestTreadmarksApiTestLock(t *testing.T) {
	tm0, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()
	tm3, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm3.Initialize(1003)
	tm3.Join("localhost", 1000)
	defer tm3.Shutdown()

	//go0:= make(chan bool, 1)
	//go1 := make(chan bool, 1)
	//go2 :=  make(chan bool, 1)
	//go3 :=  make(chan bool, 1)

	var lockId uint8 = 0

	lock0, lock1, lock2, lock3 := tm0.locks[lockId], tm1.locks[lockId], tm2.locks[lockId], tm3.locks[lockId]

	go func() {
		tm1.AcquireLock(lockId)
	}()
	time.Sleep(time.Millisecond * 100)
	go func() {
		tm2.AcquireLock(lockId)
	}()
	time.Sleep(time.Millisecond * 100)
	go func() {
		tm3.AcquireLock(lockId)
	}()
	time.Sleep(time.Millisecond * 100)

	assert.False(t, lock0.locked)
	assert.True(t, lock1.locked)
	assert.False(t, lock2.locked)
	assert.False(t, lock3.locked)
	assert.False(t, lock0.haveToken)
	assert.True(t, lock1.haveToken)
	assert.False(t, lock2.haveToken)
	assert.False(t, lock3.haveToken)
	assert.Equal(t, byte(3), lock0.last)
	assert.Equal(t, byte(2), lock1.nextId)
	assert.Equal(t, byte(3), lock2.nextId)
	assert.Equal(t, byte(0), lock3.nextId)
	assert.Nil(t, lock0.nextTimestamp)
	assert.NotNil(t, lock1.nextTimestamp)
	assert.NotNil(t, lock2.nextTimestamp)
	assert.Nil(t, lock3.nextTimestamp)

	tm1.ReleaseLock(lockId)

	time.Sleep(time.Millisecond * 100)

	assert.False(t, lock0.locked)
	assert.False(t, lock1.locked)
	assert.True(t, lock2.locked)
	assert.False(t, lock3.locked)
	assert.False(t, lock0.haveToken)
	assert.False(t, lock1.haveToken)
	assert.True(t, lock2.haveToken)
	assert.False(t, lock3.haveToken)
	assert.Equal(t, byte(3), lock0.last)
	assert.Equal(t, byte(0), lock1.nextId)
	assert.Equal(t, byte(3), lock2.nextId)
	assert.Equal(t, byte(0), lock3.nextId)
	assert.Nil(t, lock0.nextTimestamp)
	assert.Nil(t, lock1.nextTimestamp)
	assert.NotNil(t, lock2.nextTimestamp)
	assert.Nil(t, lock3.nextTimestamp)

	tm2.ReleaseLock(lockId)

	time.Sleep(time.Millisecond * 100)

	assert.False(t, lock0.locked)
	assert.False(t, lock1.locked)
	assert.False(t, lock2.locked)
	assert.True(t, lock3.locked)
	assert.False(t, lock0.haveToken)
	assert.False(t, lock1.haveToken)
	assert.False(t, lock2.haveToken)
	assert.True(t, lock3.haveToken)
	assert.Equal(t, byte(3), lock0.last)
	assert.Equal(t, byte(0), lock1.nextId)
	assert.Equal(t, byte(0), lock2.nextId)
	assert.Equal(t, byte(0), lock3.nextId)
	assert.Nil(t, lock0.nextTimestamp)
	assert.Nil(t, lock1.nextTimestamp)
	assert.Nil(t, lock2.nextTimestamp)
	assert.Nil(t, lock3.nextTimestamp)
}

func TestTreadmarksApiTestLock2(t *testing.T) {
	tm0, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm0.Initialize(1000)
	tm1, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	tm2, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	tm3, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm3.Initialize(1003)
	tm3.Join("localhost", 1000)
	defer tm0.Shutdown()
	defer tm1.Shutdown()
	defer tm2.Shutdown()
	defer tm3.Shutdown()

	//go0:= make(chan bool, 1)
	//go1 := make(chan bool, 1)
	//go2 :=  make(chan bool, 1)
	//go3 :=  make(chan bool, 1)

	var lockId uint8 = 0

	lock0, lock1, lock2, lock3 := tm0.locks[lockId], tm1.locks[lockId], tm2.locks[lockId], tm3.locks[lockId]

	tm1.AcquireLock(lockId)
	tm1.ReleaseLock(lockId)
	time.Sleep(time.Millisecond * 100)

	assert.False(t, lock0.locked)
	assert.False(t, lock1.locked)
	assert.False(t, lock2.locked)
	assert.False(t, lock3.locked)
	assert.False(t, lock0.haveToken)
	assert.True(t, lock1.haveToken)
	assert.False(t, lock2.haveToken)
	assert.False(t, lock3.haveToken)
	assert.Equal(t, byte(1), lock0.last)
	assert.Equal(t, byte(0), lock1.nextId)
	assert.Equal(t, byte(0), lock2.nextId)
	assert.Equal(t, byte(0), lock3.nextId)
	assert.Nil(t, lock0.nextTimestamp)
	assert.Nil(t, lock1.nextTimestamp)
	assert.Nil(t, lock2.nextTimestamp)
	assert.Nil(t, lock3.nextTimestamp)

	tm2.AcquireLock(lockId)
	tm2.ReleaseLock(lockId)

	time.Sleep(time.Millisecond * 100)

	assert.False(t, lock0.locked)
	assert.False(t, lock1.locked)
	assert.False(t, lock2.locked)
	assert.False(t, lock3.locked)
	assert.False(t, lock0.haveToken)
	assert.False(t, lock1.haveToken)
	assert.True(t, lock2.haveToken)
	assert.False(t, lock3.haveToken)
	assert.Equal(t, byte(2), lock0.last)
	assert.Equal(t, byte(0), lock1.nextId)
	assert.Equal(t, byte(0), lock2.nextId)
	assert.Equal(t, byte(0), lock3.nextId)
	assert.Nil(t, lock0.nextTimestamp)
	assert.Nil(t, lock1.nextTimestamp)
	assert.Nil(t, lock2.nextTimestamp)
	assert.Nil(t, lock3.nextTimestamp)

	tm3.AcquireLock(lockId)
	tm3.ReleaseLock(lockId)

	time.Sleep(time.Millisecond * 100)

	assert.False(t, lock0.locked)
	assert.False(t, lock1.locked)
	assert.False(t, lock2.locked)
	assert.False(t, lock3.locked)
	assert.False(t, lock0.haveToken)
	assert.False(t, lock1.haveToken)
	assert.False(t, lock2.haveToken)
	assert.True(t, lock3.haveToken)
	assert.Equal(t, byte(3), lock0.last)
	assert.Equal(t, byte(0), lock1.nextId)
	assert.Equal(t, byte(0), lock2.nextId)
	assert.Equal(t, byte(0), lock3.nextId)
	assert.Nil(t, lock0.nextTimestamp)
	assert.Nil(t, lock1.nextTimestamp)
	assert.Nil(t, lock2.nextTimestamp)
	assert.Nil(t, lock3.nextTimestamp)
}

func TestTreadmarksApi_Barriers(t *testing.T) {
	tm0, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm0.Initialize(1000)
	tm1, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	tm2, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	tm3, _ := NewTreadmarksApi(1024, 128, 4, 3, 3)
	tm3.Initialize(1003)
	tm3.Join("localhost", 1000)

	defer tm0.Shutdown()
	defer tm1.Shutdown()
	defer tm2.Shutdown()
	defer tm3.Shutdown()

	go0 := make(chan bool)
	go1 := make(chan bool)
	go2 := make(chan bool)
	go3 := make(chan bool)

	test := func(tm *TreadmarksApi, g chan bool) {
		go func() {
			tm.Write(0, byte(0))
			tm.Barrier(0)
			g <- true
			<-g
			tm.Write(0, byte(0))
		}()
		time.Sleep(time.Millisecond * 100)
	}

	test(tm0, go0)
	test(tm1, go1)
	test(tm2, go2)

	assert.True(t, timeout(go0))
	assert.True(t, timeout(go1))
	assert.True(t, timeout(go2))

	test(tm3, go3)

	assert.False(t, timeout(go0))
	assert.False(t, timeout(go1))
	assert.False(t, timeout(go2))
	assert.False(t, timeout(go3))
	go0 <- true
	go1 <- true
	go2 <- true
	go3 <- true
	time.Sleep(time.Millisecond * 100)

	test(tm1, go1)
	test(tm0, go0)
	test(tm2, go2)

	assert.True(t, timeout(go1))
	assert.True(t, timeout(go0))
	assert.True(t, timeout(go2))

	test(tm3, go3)

	assert.False(t, timeout(go0))
	assert.False(t, timeout(go1))
	assert.False(t, timeout(go2))
	assert.False(t, timeout(go3))
	go0 <- true
	go1 <- true
	go2 <- true
	go3 <- true
	time.Sleep(time.Millisecond * 100)

	test(tm1, go1)
	test(tm2, go2)
	test(tm0, go0)

	assert.True(t, timeout(go1))
	assert.True(t, timeout(go0))
	assert.True(t, timeout(go2))

	test(tm3, go3)
	assert.False(t, timeout(go0))
	assert.False(t, timeout(go1))
	assert.False(t, timeout(go2))
	assert.False(t, timeout(go3))
	go0 <- true
	go1 <- true
	go2 <- true
	go3 <- true
	time.Sleep(time.Millisecond * 100)

	test(tm1, go1)
	test(tm2, go2)
	test(tm3, go3)

	assert.True(t, timeout(go1))
	assert.True(t, timeout(go0))
	assert.True(t, timeout(go2))

	test(tm0, go0)
	assert.False(t, timeout(go0))
	assert.False(t, timeout(go1))
	assert.False(t, timeout(go2))
	assert.False(t, timeout(go3))

}

func TestNewTreadmarksApi_LockIntervalsMultipleProcs(t *testing.T) {
	tm0, _ := NewTreadmarksApi(30, 10, 3, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadmarksApi(30, 10, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadmarksApi(30, 10, 3, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	tm0.AcquireLock(0)
	tm0.Write(0, 4)
	tm0.ReleaseLock(0)
	tm1.AcquireLock(0)
	tm1.Write(0, 5)
	tm1.ReleaseLock(0)
	tm0.AcquireLock(0)
	tm0.Write(0, 6)
	tm0.ReleaseLock(0)
	tm1.AcquireLock(0)
	tm1.Write(0, 7)
	tm1.ReleaseLock(0)
	tm0.AcquireLock(0)
	tm0.ReleaseLock(0)
	tm2.AcquireLock(0)
	val, err := tm2.Read(0)
	assert.Equal(t, byte(7), val)
	assert.Nil(t, err)
	assert.Len(t, tm0.procarray[0], 2)
	assert.Len(t, tm1.procarray[0], 2)
	assert.Len(t, tm2.procarray[0], 2)
	assert.Len(t, tm0.procarray[1], 2)
	assert.Len(t, tm1.procarray[1], 2)
	assert.Len(t, tm2.procarray[1], 2)
	assert.Len(t, tm0.procarray[2], 0)
	assert.Len(t, tm1.procarray[2], 0)
	assert.Len(t, tm2.procarray[2], 0)

	assert.Len(t, tm0.pagearray[0].writenotices[0], 2)
	assert.Len(t, tm1.pagearray[0].writenotices[0], 2)
	assert.Len(t, tm2.pagearray[0].writenotices[0], 2)
	assert.Len(t, tm0.pagearray[0].writenotices[1], 2)
	assert.Len(t, tm1.pagearray[0].writenotices[1], 2)
	assert.Len(t, tm2.pagearray[0].writenotices[1], 2)
	assert.Len(t, tm0.pagearray[0].writenotices[2], 0)
	assert.Len(t, tm1.pagearray[0].writenotices[2], 0)
	assert.Len(t, tm2.pagearray[0].writenotices[2], 0)
}

func TestNewTreadmarksApi_LockIntervalsMultipleProcs_2(t *testing.T) {
	tm0, _ := NewTreadmarksApi(30, 10, 3, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadmarksApi(30, 10, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	tm2, _ := NewTreadmarksApi(30, 10, 3, 3, 3)
	tm2.Initialize(1002)
	tm2.Join("localhost", 1000)
	defer tm2.Shutdown()

	tm0.AcquireLock(0)
	tm0.Write(1, 10)
	tm0.ReleaseLock(0)
	tm0.AcquireLock(0)
	tm0.ReleaseLock(0)
	tm1.AcquireLock(0)
	val, _ := tm1.Read(1)
	assert.Equal(t, byte(10), val)
	fmt.Println("boom")
	tm0.Write(2, 11)
	tm1.ReleaseLock(0)
	tm0.AcquireLock(0)
	val, _ = tm0.Read(2)
	assert.Equal(t, byte(11), val)
	tm0.ReleaseLock(0)
	tm1.AcquireLock(0)
	val, _ = tm1.Read(2)
	assert.Equal(t, byte(11), val)
	tm1.ReleaseLock(0)
	tm0.Write(12, 10)
	tm0.Write(24, 11)
	tm0.AcquireLock(0)
	tm0.ReleaseLock(0)
	tm1.AcquireLock(0)
	val, _ = tm1.Read(12)
	assert.Equal(t, byte(10), val)
	val, _ = tm1.Read(24)
	assert.Equal(t, byte(11), val)
}

func TestTreadmarksApi_RepeatedLock(t *testing.T) {
	tm0, _ := NewTreadmarksApi(30, 10, 3, 3, 3)
	tm0.Initialize(1000)
	defer tm0.Shutdown()
	tm1, _ := NewTreadmarksApi(30, 10, 3, 3, 3)
	tm1.Initialize(1001)
	tm1.Join("localhost", 1000)
	defer tm1.Shutdown()
	runtime.GOMAXPROCS(4)
	go0, go1 := make(chan int), make(chan int)
	i := 0
	go func() {
		tm := tm0
		g := go0
		<-g
		time.Sleep(time.Millisecond * 100)
		fmt.Println("0")
		for {

			tm.AcquireLock(0)

			val, _ := tm.Read(0)
			tm.Write(0, val+1%256)
			tm.ReleaseLock(0)
			if val > 253 {
				fmt.Println("0, ", i)
				i++
				if i > 1000 {
					fmt.Println("boom")
					g <- 1
					return
				}
			}
		}

	}()
	go func() {
		tm := tm1
		g := go1
		<-g
		time.Sleep(time.Millisecond * 100)
		fmt.Println("1")
		for {
			tm.AcquireLock(0)

			val, _ := tm.Read(0)
			tm.Write(0, val+1%256)
			tm.ReleaseLock(0)
			if val > 253 {
				fmt.Println("1, ", i)
				i++
				if i > 1000 {
					fmt.Println("boom")
					g <- 1
					return
				}
			}
		}

	}()
	go0 <- 1
	go1 <- 2
	<-go1
	<-go0

}

func readInt(dsm dsm_api.DSMApiInterface, addr int) int {
	bInt := make([]byte, 4)
	var err error
	for i := range bInt {
		bInt[i], err = dsm.Read(addr + i)
		if err != nil {
			panic(err.Error())
		}
	}
	output := int(utils.BytesToInt32(bInt))
	fmt.Println("read - ", addr, " - ", output, " - ", bInt)
	return output
}

func writeInt(dsm dsm_api.DSMApiInterface, addr, input int) {
	bInt := utils.Int32ToBytes(int32(input))
	fmt.Println("write - ", addr, " - ", input, " - ", bInt)
	var err error
	for i := range bInt {
		err = dsm.Write(addr+i, bInt[i])
		if err != nil {
			panic(err.Error())
		}
	}
}

func timeout(channel chan bool) bool {
	select {
	case <-channel:
		return false
	case <-time.After(time.Millisecond * 100):
		return true
	}
}
