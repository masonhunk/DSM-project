package treadmarks

import (
	"DSM-project/network"
	"net"
)

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

func NewTM_Manager() *tm_Manager {
	m := new(tm_Manager)
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
