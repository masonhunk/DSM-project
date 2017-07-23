package treadmarks

import (
	"DSM-project/memory"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"sync"
	"testing"
	"time"
)

var _ = fmt.Print
var _ = log.Print

func setupTreadMarksStruct(nrProcs int) *TreadMarks {
	vm1 := memory.NewVmem(128, 8)
	tm1 := NewTreadMarks(vm1, nrProcs, 4, 4)
	return tm1
}

func TestTreadMarksInitialisation(t *testing.T) {

	managerHost := setupTreadMarksStruct(3)
	host2 := setupTreadMarksStruct(3)
	host3 := setupTreadMarksStruct(3)
	err := managerHost.Startup()
	assert.Nil(t, err)
	err = host2.Join("localhost:2000")
	assert.Nil(t, err)
	err = host3.Join("localhost:2000")
	assert.Nil(t, err)
	assert.NotNil(t, managerHost)

	assert.Equal(t, byte(1), managerHost.ProcId)
	assert.Equal(t, byte(2), host2.ProcId)
	assert.Equal(t, byte(3), host3.ProcId)

	host2.Shutdown()
	host3.Shutdown()
	managerHost.Shutdown()

}

func TestBarrier(t *testing.T) {
	managerHost := setupTreadMarksStruct(3)
	host2 := setupTreadMarksStruct(3)
	host3 := setupTreadMarksStruct(3)
	managerHost.Startup()
	host2.Join("localhost:2000")
	host3.Join("localhost:2000")

	started := make(chan bool, 3)
	done := make(chan bool)

	go func(host *TreadMarks, started chan<- bool, done chan<- bool) {
		started <- true
		host.Barrier(1)
		done <- true
	}(managerHost, started, done)
	<-started
	var failed bool
	select {
	case <-done:
		failed = true
	default:
		failed = false
	}
	assert.False(t, failed)

	go func(host *TreadMarks, started chan<- bool, done chan<- bool) {
		started <- true
		host.Barrier(1)
		done <- true
	}(host2, started, done)
	<-started
	select {
	case <-done:
		failed = true
	default:
		failed = false
	}
	assert.False(t, failed)

	go func(host *TreadMarks, started chan<- bool, done chan<- bool) {
		started <- true
		host.Barrier(1)
		done <- true
	}(host3, started, done)
	<-started

	<-done
	<-done
	<-done

	//add tests of validity of data structures

	host2.Shutdown()
	host3.Shutdown()
	managerHost.Shutdown()

}

func TestLocks(t *testing.T) {
	managerHost := setupTreadMarksStruct(3)
	host2 := setupTreadMarksStruct(3)
	host3 := setupTreadMarksStruct(3)
	managerHost.Startup()
	host2.Join("localhost:2000")
	host3.Join("localhost:2000")
	started := make(chan string)
	finished := make(chan string)

	go func() {
		started <- "ok"
		host2.AcquireLock(1)
		time.Sleep(500 * time.Millisecond)
		host2.ReleaseLock(1)
		finished <- "released"
	}()
	<-started
	time.Sleep(200 * time.Millisecond)
	go func() {
		started <- "ok"
		host3.AcquireLock(1)
		finished <- "acquired"
	}()
	<-started
	assert.Equal(t, "released", <-finished)
	assert.Equal(t, "acquired", <-finished)
}

func TestShouldGetCopyIfNoCopy(t *testing.T) {
	managerHost := setupTreadMarksStruct(2)
	host2 := setupTreadMarksStruct(2)
	managerHost.Startup()
	host2.Join("localhost:2000")
	assert.False(t, managerHost.GetPageEntry(1).hascopy)

	managerHost.Write(13, byte(10))

	assert.True(t, managerHost.GetPageEntry(1).hascopy)
	assert.Equal(t, memory.READ_WRITE, managerHost.GetRights(13))

	res, _ := managerHost.Read(13)

	assert.Equal(t, byte(10), res)
	assert.False(t, host2.GetPageEntry(1).hascopy)

	res, _ = host2.Read(13) //in page 1

	assert.True(t, host2.GetPageEntry(1).hascopy)
	assert.Equal(t, byte(10), res)
	assert.Equal(t, memory.READ_ONLY, host2.GetRights(13))
}

func TestBarrierManagerVCUpdate(t *testing.T) {
	host1 := setupTreadMarksStruct(3)
	host2 := setupTreadMarksStruct(3)
	host3 := setupTreadMarksStruct(3)

	host1.Startup()
	host2.Join("localhost:2000")
	host3.Join("localhost:2000")

	assert.Equal(t, *NewVectorclock(4), host1.vc)
	assert.Equal(t, *NewVectorclock(4), host2.vc)
	assert.Equal(t, *NewVectorclock(4), host3.vc)

	host1.AcquireLock(1)
	host1.ReleaseLock(1)
	host2.AcquireLock(1)
	host2.ReleaseLock(1)
	host3.AcquireLock(1)
	host3.ReleaseLock(1)
	time.Sleep(time.Millisecond * 300)
	assert.Equal(t, Vectorclock{Value: []uint{0, 1, 0, 0}}, host1.vc)
	assert.Equal(t, Vectorclock{Value: []uint{0, 1, 1, 0}}, host2.vc)
	assert.Equal(t, Vectorclock{Value: []uint{0, 1, 1, 0}}, host3.vc)

	done := make(chan bool)
	go func() {
		host1.Barrier(1)
		done <- true
	}()
	go func() {
		host2.Barrier(1)
		done <- true
	}()
	go func() {
		host3.Barrier(1)
		done <- true
	}()
	<-done
	<-done
	<-done
	assert.Equal(t, Vectorclock{Value: []uint{1, 1, 1, 0}}, host1.vc)
	assert.Equal(t, Vectorclock{Value: []uint{1, 1, 1, 0}}, host2.vc)
	assert.Equal(t, Vectorclock{Value: []uint{1, 1, 1, 0}}, host3.vc)

}

func TestCreationAndPropagationOfWriteNotices(t *testing.T) {
	host1 := setupTreadMarksStruct(3)
	host2 := setupTreadMarksStruct(3)
	host3 := setupTreadMarksStruct(3)

	host1.Startup()
	host2.Join("localhost:2000")
	host3.Join("localhost:2000")

	host1.AcquireLock(1)
	host1.Write(13, byte(10))
	host1.ReleaseLock(1)
	host2.AcquireLock(1)

	assert.Equal(t, Vectorclock{Value: []uint{0, 1, 0, 0}}, host1.GetIntervalRecordHead(byte(1)).Timestamp)
	assert.Len(t, host1.GetIntervalRecordHead(byte(1)).WriteNotices, 1)
	assert.Equal(t, Vectorclock{Value: []uint{0, 1, 0, 0}}, host2.vc)

	host2.Write(25, byte(8))
	host2.Write(13, byte(12))
	host2.Write(0, byte(1))
	host2.ReleaseLock(1)

	host3.AcquireLock(1)

	assert.Equal(t, Vectorclock{Value: []uint{0, 1, 0, 0}}, host2.GetIntervalRecordHead(byte(1)).Timestamp)
	assert.Equal(t, Vectorclock{Value: []uint{0, 1, 1, 0}}, host2.GetIntervalRecordHead(byte(2)).Timestamp)
	assert.Len(t, host2.GetIntervalRecordHead(byte(1)).WriteNotices, 1)
	assert.Len(t, host2.GetIntervalRecordHead(byte(2)).WriteNotices, 3)
	assert.Len(t, host2.GetWritenoticeList(byte(1), 1), 1)
	assert.Len(t, host2.GetWritenoticeList(byte(2), 0), 1)
	assert.Len(t, host2.GetWritenoticeList(byte(2), 1), 1)
	assert.Len(t, host2.GetWritenoticeList(byte(2), 3), 1)
	assert.Len(t, host2.GetWritenoticeList(byte(1), 0), 0)

	host3.ReleaseLock(1)

	res, _ := host1.Read(13)
	assert.Equal(t, byte(10), res)
	res1, _ := host2.Read(13)
	res2, _ := host2.Read(0)
	res3, _ := host2.Read(25)

	//host 2 should see all its own changes
	assert.Equal(t, byte(12), res1)
	assert.Equal(t, byte(1), res2)
	assert.Equal(t, byte(8), res3)

	//host 3 should see all changes by host 2
	res1, _ = host3.Read(13)
	res2, _ = host3.Read(0)
	res3, _ = host3.Read(25)

	assert.Equal(t, byte(12), res1)
	assert.Equal(t, byte(1), res2)
	assert.Equal(t, byte(8), res3)
}

func TestWritesBeforeAcquire(t *testing.T) {
	host1 := setupTreadMarksStruct(3)
	host2 := setupTreadMarksStruct(3)
	host3 := setupTreadMarksStruct(3)

	host1.Startup()
	host2.Join("localhost:2000")
	host3.Join("localhost:2000")

	host2.Write(0, byte(5))
	val, _ := host2.Read(0)
	assert.Equal(t, byte(5), val)

	val, _ = host3.Read(0)
	assert.Equal(t, byte(0), val)

	host3.AcquireLock(1)
	val, _ = host3.Read(0)
	assert.Equal(t, byte(0), val)
	host3.Write(1, byte(2))
	host3.ReleaseLock(1)
	time.Sleep(time.Second)
	host2.AcquireLock(1)
	val, _ = host2.Read(0)
	assert.Equal(t, byte(5), val)
	val, _ = host2.Read(1)
	assert.Equal(t, byte(2), val)

}

func TestShouldNotCreateNewIntervalOnLockReacquire(t *testing.T) {

}

func TestBarrierReadWrites(t *testing.T) {
	host1 := setupTreadMarksStruct(3)
	host2 := setupTreadMarksStruct(3)
	host3 := setupTreadMarksStruct(3)

	host1.Startup()
	host2.Join("localhost:2000")
	host3.Join("localhost:2000")

	started := make(chan bool, 1)
	group := new(sync.WaitGroup)
	group.Add(3)
	go func() {
		started <- true
		host1.AcquireLock(1)
		host1.Write(13, byte(13))
		host1.ReleaseLock(1)
		host1.Barrier(1)
		group.Done()
	}()
	<-started
	go func() {
		started <- true
		host2.AcquireLock(2)
		host2.Write(12, byte(12))
		host2.ReleaseLock(2)
		host2.Barrier(1)
		group.Done()
	}()
	<-started
	go func() {
		started <- true
		host3.Write(1, byte(1))
		host3.Barrier(1)
		group.Done()
	}()
	<-started
	group.Wait()


	assert.Equal(t, Vectorclock{Value: []uint{1, 1, 1, 1}}, host1.vc)
	assert.Equal(t, Vectorclock{Value: []uint{1, 1, 1, 1}}, host2.vc)
	assert.Equal(t, Vectorclock{Value: []uint{1, 1, 1, 1}}, host3.vc)

	assert.Equal(t, Vectorclock{Value: []uint{0, 0, 1, 0}}, host1.GetWritenoticeList(byte(2), 1)[0].Interval.Timestamp)
	//all changes made in host1 and host2 should be seen by all.
	res1, _ := host1.Read(12)
	res2, _ := host1.Read(13)
	assert.Equal(t, byte(12), res1) //failed
	assert.Equal(t, byte(13), res2)

	/*	res1, _ = host2.Read(12)
		res2, _ = host2.Read(13)
		assert.Equal(t, byte(12), res1)
		assert.Equal(t, byte(13), res2) //failed

		res1, _ = host3.Read(12)
		res2, _ = host3.Read(13)
		assert.Equal(t, byte(12), res1)
		assert.Equal(t, byte(13), res2)

		// The write by host3 should not be seen since it was not part of a lock
		res1, _ = host1.Read(1)
		res2, _ = host2.Read(1)
		assert.Equal(t, byte(0), res1)
		assert.Equal(t, byte(0), res2)*/

}
