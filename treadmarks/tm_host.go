package treadmarks

import (
	"DSM-project/memory"
	"DSM-project/network"
)

type ITreadMarks interface {
	memory.VirtualMemory
	Startup() error
	Shutdown()
	AcquireLock(id int)
	ReleaseLock(id int)
	barrier(id int)
}

type TreadMarks struct {
	memory.VirtualMemory //embeds this interface type
	nrProcs              int
	procId               int
	nrLocks              int
	nrBarriers           int
	network.IClient
}

func NewTreadMarks(client network.IClient, virtualMemory memory.VirtualMemory) {
	//TODO: setup virtual memory onFault listener
	panic("implement me")
}

func (t *TreadMarks) Startup() error {
	panic("implement me")
}

func (t *TreadMarks) Shutdown() {
	panic("implement me")
}

func (t *TreadMarks) AcquireLock(id int) {
	panic("implement me")
}

func (t *TreadMarks) ReleaseLock(id int) {
	panic("implement me")
}

func (t *TreadMarks) barrier(id int) {
	panic("implement me")
}
