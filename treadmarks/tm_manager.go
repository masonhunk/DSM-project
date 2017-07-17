package treadmarks

import (
	"DSM-project/network"
	"errors"
	"net"
	"sync"
)

//Interfaces
type LockManager interface {
	HandleLockAcquire(id int)
	HandleLockRelease(id int) error
}

type BarrierManager interface {
	HandleBarrier(id int)
}

//Lock manager implementation
type LockManagerImp struct {
	locks   map[int]*sync.Mutex
	history map[int]int
	*sync.Mutex
}

func NewLockManagerImp() *LockManagerImp {
	lm := new(LockManagerImp)
	lm.locks = make(map[int]*sync.Mutex)
	lm.Mutex = new(sync.Mutex)
	return lm
}

func (lm *LockManagerImp) HandleLockAcquire(id int) {
	lm.Lock()
	lock, ok := lm.locks[id]
	if ok == false {
		lock = new(sync.Mutex)
		lm.locks[id] = lock
	}
	lm.Unlock()
	lock.Lock()
}

func (lm *LockManagerImp) HandleLockRelease(id int) error {
	lm.Lock()
	lock, ok := lm.locks[id]
	if ok == false {
		return errors.New("LockManager doesn't have a lock with ID " + string(id))
	}
	lm.Unlock()
	lock.Unlock()
	return nil
}

//Barrier manager implementation
type BarrierManagerImp struct {
	barriers map[int]*sync.WaitGroup
	nodes    int
	*sync.Mutex
}

func NewBarrierManagerImp(nodes int) *BarrierManagerImp {
	bm := new(BarrierManagerImp)
	bm.nodes = nodes
	bm.barriers = make(map[int]*sync.WaitGroup)
	bm.Mutex = new(sync.Mutex)

	return bm
}

func (bm *BarrierManagerImp) HandleBarrier(id int) {
	bm.Lock()
	barrier, ok := bm.barriers[id]
	if ok == false {
		barrier = new(sync.WaitGroup)
		barrier.Add(bm.nodes)
		bm.barriers[id] = barrier
	}
	bm.Unlock()
	barrier.Done()
	barrier.Wait()
}

type tm_Manager struct {
	BarrierManager
	LockManager
	network.ITransciever //embedded type
}

func NewTM_Manager(bm BarrierManager, lm LockManager) *tm_Manager {
	m := new(tm_Manager)
	m.BarrierManager = bm
	m.LockManager = lm
	return m
}

func (m *tm_Manager) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	m.ITransciever = network.NewTransciever(conn, func(message network.Message) error {

		return nil
	})

	return nil
}
