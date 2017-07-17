package tests

import (
	"testing"
	"DSM-project/treadmarks"

	"github.com/stretchr/testify/assert"
)

//First we test the lock manager

//Here we test if the lock manager can handle getting locked and such.
func TestLockManagerCreation(t *testing.T) {
	var lm treadmarks.LockManager
	lm = treadmarks.NewLockManagerImp()
	lm.HandleLockAcquire(2)
	lm.HandleLockAcquire(3)
	lm.HandleLockRelease(2)
	lm.HandleLockRelease(3)
}

//Here we test that when something is locked, you cant access that lock.
func TestLockManagerOrder(t *testing.T){
	var lm treadmarks.LockManager
	lm = treadmarks.NewLockManagerImp()
	order := make(chan int, 5)
	go func() {
		lm.HandleLockAcquire(1)
		order <- 1
	}()
	assert.Equal(t, 1, <- order)
	go func() {
		lm.HandleLockAcquire(1)
		order <- 2
	}()

	lm.HandleLockRelease(1)
	assert.Equal(t, 2, <- order)
	lm.HandleLockRelease(1)
}

func TestBarrierManager(t *testing.T){

}