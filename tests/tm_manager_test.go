package tests

import (
	"DSM-project/network"
	"DSM-project/treadmarks"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

//This is a mock transciever that just saves the messages.
type TranscieverMock struct {
	messages []treadmarks.TM_Message
}

func NewTranscieverMock(messages []treadmarks.TM_Message) *TranscieverMock {
	tr := new(TranscieverMock)
	tr.messages = messages
	return tr
}

func (t *TranscieverMock) Close() {
	panic("implement me")
}

func (t *TranscieverMock) Send(message network.Message) error {
	t.messages = append(t.messages, message.(treadmarks.TM_Message))
	return nil
}

//First we test the lock manager

//Here we test if the lock manager can handle getting locked and such.
func TestLockManagerCreation(t *testing.T) {
	fmt.Println()
	var lm treadmarks.LockManager
	lm = treadmarks.NewLockManagerImp()
	id1 := lm.HandleLockAcquire(2)
	id2 := lm.HandleLockAcquire(3)
	lm.HandleLockRelease(2, byte(1))
	lm.HandleLockRelease(3, byte(2))

	assert.Equal(t, byte(0), id1, "Former owner should be 0.")
	assert.Equal(t, byte(0), id2, "Former owner should be 0.")
}

//Here we test that when something is locked, you cant access that lock.
func TestLockManagerOrder(t *testing.T) {
	var lm treadmarks.LockManager
	lm = treadmarks.NewLockManagerImp()
	id1 := byte(0)
	id2 := byte(0)
	order := make(chan int, 5)
	go func() {
		id1 = lm.HandleLockAcquire(1)
		order <- 1
	}()
	assert.Equal(t, 1, <-order, "The first go-routine should be allowed get the lock.")
	go func() {
		id2 = lm.HandleLockAcquire(1)
		order <- 2
	}()

	lm.HandleLockRelease(1, 1)
	assert.Equal(t, 2, <-order, "At this point, the second go routine should get the lock.")
	lm.HandleLockRelease(1, 2)
	assert.Equal(t, byte(0), id1, "The first goroutine should see 0 as the previous owner.")
	assert.Equal(t, byte(1), id2, "The second go-routine should see 1 as the previous owner.")
}

func TestBarrierManager(t *testing.T) {
	var bm treadmarks.BarrierManager
	bm = treadmarks.NewBarrierManagerImp(3)
	order := make(chan int)
	done := make(chan bool, 5)
	go func() {
		order <- 1
		bm.HandleBarrier(1)
		done <- true
	}()
	assert.Equal(t, 1, <-order, "The first go-routine must be waiting by now.")
	assert.Equal(t, 0, len(done), "None of the go-routines can be finished yet.")
	go func() {
		order <- 2
		bm.HandleBarrier(1)
		done <- true
	}()
	assert.Equal(t, 2, <-order, "The second go-routine must be waiting by now.")
	assert.Equal(t, 0, len(done), "None of the go-routines can be finished yet.")
	go func() {
		order <- 3
		bm.HandleBarrier(1)
		done <- true
	}()
	assert.Equal(t, 0, len(done), "None of the go-routines can be finished yet.")
	assert.Equal(t, 3, <-order, "The third goroutine should be waiting for waitgroup.")
	assert.Equal(t, true, <-done, "All the goroutines should be finished by now.")
	assert.Equal(t, true, <-done, "All the goroutines should be finished by now.")
	assert.Equal(t, true, <-done, "All the goroutines should be finished by now.")
}

func TestManagerHandleLockAcquireNotExistMessage(t *testing.T) {
	messages := make([]treadmarks.TM_Message, 0)
	tr := NewTranscieverMock(messages)
	bm := treadmarks.NewBarrierManagerImp(4)
	lm := treadmarks.NewLockManagerImp()
	m := treadmarks.NewTest_TM_Manager(tr, bm, lm, 4, nil)
	m.HandleMessage(treadmarks.TM_Message{Type: treadmarks.LOCK_ACQUIRE_REQUEST, Id: 1, From: byte(1), To: byte(2)})
	assert.Equal(t, treadmarks.TM_Message{Type: treadmarks.LOCK_ACQUIRE_RESPONSE, To: byte(1), Id: 1, From: byte(0)}, tr.messages[0])
}

func TestManagerHandleLockAcquireExistsMessage(t *testing.T) {
	messages := make([]treadmarks.TM_Message, 0)
	tr := NewTranscieverMock(messages)
	bm := treadmarks.NewBarrierManagerImp(4)
	lm := treadmarks.NewLockManagerImp()
	m := treadmarks.NewTest_TM_Manager(tr, bm, lm, 4, nil)
	m.HandleMessage(treadmarks.TM_Message{Type: treadmarks.LOCK_ACQUIRE_REQUEST, Id: 1, From: byte(1), To: byte(2)})
	assert.Equal(t, treadmarks.TM_Message{Type: treadmarks.LOCK_ACQUIRE_RESPONSE, To: byte(1), Id: 1, From: byte(0)}, tr.messages[0])
	m.HandleMessage(treadmarks.TM_Message{Type: treadmarks.LOCK_RELEASE, Id: 1, From: byte(1), To: byte(2)})
	m.HandleMessage(treadmarks.TM_Message{Type: treadmarks.LOCK_ACQUIRE_REQUEST, Id: 1, From: byte(3), To: byte(2)})
	assert.Equal(t, treadmarks.TM_Message{Type: treadmarks.LOCK_ACQUIRE_REQUEST, To: byte(1), Id: 1, From: byte(3)}, tr.messages[1])
}

func TestManagerHandleLockReleaseMessageError(t *testing.T) {
	messages := make([]treadmarks.TM_Message, 0)
	tr := NewTranscieverMock(messages)
	bm := treadmarks.NewBarrierManagerImp(4)
	lm := treadmarks.NewLockManagerImp()
	m := treadmarks.NewTest_TM_Manager(tr, bm, lm, 4, nil)
	err := m.HandleMessage(treadmarks.TM_Message{Type: treadmarks.LOCK_RELEASE, Id: 1, From: byte(1), To: byte(2)})
	assert.EqualError(t, err, "LockManager doesn't have a lock with ID 1", "Since this lock has"+
		"not been locked before, it should give an error when unlocked.")
	assert.Equal(t, 0, len(tr.messages))
}

func TestManagerHandleBarrierMessage(t *testing.T) {
	messages := make([]treadmarks.TM_Message, 0)
	tr := NewTranscieverMock(messages)
	bm := treadmarks.NewBarrierManagerImp(2)
	lm := treadmarks.NewLockManagerImp()
	m := treadmarks.NewTest_TM_Manager(tr, bm, lm, 4, nil)
	started := make(chan bool)
	done := make(chan bool, 2)
	go func() {
		started <- true
		m.HandleMessage(treadmarks.TM_Message{Type: treadmarks.BARRIER_REQUEST, Id: 1, From: byte(1), To: byte(0)})
		done <- true
	}()
	go func() {
		started <- true
		m.HandleMessage(treadmarks.TM_Message{Type: treadmarks.BARRIER_REQUEST, Id: 1, From: byte(2), To: byte(0)})
		done <- true
	}()
	<-started
	assert.Len(t, tr.messages, 0, "When only one is started, we should have sent any messages.")
	<-started
	<-done
	<-done
	assert.Contains(t, tr.messages, treadmarks.TM_Message{Type: treadmarks.BARRIER_RESPONSE, Id: 1, From: byte(0), To: byte(1)})
	assert.Contains(t, tr.messages, treadmarks.TM_Message{Type: treadmarks.BARRIER_RESPONSE, Id: 1, From: byte(0), To: byte(2)})
}
