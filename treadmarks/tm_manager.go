package treadmarks

import "DSM-project/network"

type LockManager interface {
	handleLockAcquire(id int)
	handleLockRelease(id int)
}

type BarrierManager interface {
	handleBarrier(id int)
}

type tm_Manager struct {
	BarrierManager
	LockManager
	network.ITransciever //embedded type
}