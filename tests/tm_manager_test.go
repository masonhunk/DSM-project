package tests

import (
	"testing"
	"DSM-project/treadmarks"
	"DSM-project/network"
	"github.com/stretchr/testify/assert"
	"fmt"
)

//This is a mock transciever that just saves the messages.
type TranscieverMock struct{
	messages []treadmarks.TM_Message
}

func NewTranscieverMock(messages []treadmarks.TM_Message) *TranscieverMock{
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
func TestLockManagerOrder(t *testing.T){
	var lm treadmarks.LockManager
	lm = treadmarks.NewLockManagerImp()
	id1 := byte(0)
	id2 := byte(0)
	order := make(chan int, 5)
	go func() {
		id1 = lm.HandleLockAcquire(1)
		order <- 1
	}()
	assert.Equal(t, 1, <- order, "The first go-routine should be allowed get the lock.")
	go func() {
		id2 = lm.HandleLockAcquire(1)
		order <- 2
	}()

	lm.HandleLockRelease(1, 1)
	assert.Equal(t, 2, <- order, "At this point, the second go routine should get the lock.")
	lm.HandleLockRelease(1, 2)
	assert.Equal(t, byte(0), id1, "The first goroutine should see 0 as the previous owner.")
	assert.Equal(t, byte(1), id2, "The second go-routine should see 1 as the previous owner.")
}

func TestBarrierManager(t *testing.T){
	var bm treadmarks.BarrierManager
	bm = treadmarks.NewBarrierManagerImp(3)
	order := make(chan int)
	done := make(chan bool, 5)
	go func(){
		order <- 1
		bm.HandleBarrier(1)
		done <- true
	}()
	assert.Equal(t, 1, <- order, "The first go-routine must be waiting by now.")
	assert.Equal(t, 0, len(done), "None of the go-routines can be finished yet.")
	go func(){
		order <- 2
		bm.HandleBarrier(1)
		done <- true
	}()
	assert.Equal(t, 2, <- order, "The second go-routine must be waiting by now.")
	assert.Equal(t, 0, len(done), "None of the go-routines can be finished yet.")
	go func(){
		order <- 3
		bm.HandleBarrier(1)
		done <- true
	}()
	assert.Equal(t, 0, len(done), "None of the go-routines can be finished yet.")
	assert.Equal(t, 3, <- order, "The third goroutine should be waiting for waitgroup.")
	assert.Equal(t, true, <- done, "All the goroutines should be finished by now.")
	assert.Equal(t, true, <- done, "All the goroutines should be finished by now.")
	assert.Equal(t, true, <- done, "All the goroutines should be finished by now.")
}

func TestManagerHandleMessage(t *testing.T){
	messages := make([]treadmarks.TM_Message, 0)
	tr := NewTranscieverMock(messages)
	tr.Send(treadmarks.TM_Message{Type:"Something"})
	fmt.Println(tr.messages)
	bm := treadmarks.NewBarrierManagerImp(4)
	lm := treadmarks.NewLockManagerImp()
	m := treadmarks.NewTM_Manager(tr, bm, lm)
	m.HandleMessage(treadmarks.TM_Message{Type:treadmarks.LOCK_ACQUIRE_REQUEST, Id:1, From:1, To:2})
	assert.Equal(t, treadmarks.TM_Message{Type:treadmarks.LOCK_ACQUIRE_REQUEST, To: 0, Id:1, From:1}, tr.messages[0])
}