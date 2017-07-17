package treadmarks

import (
	"DSM-project/memory"
	"DSM-project/network"
)

const (
	LOCK_ACQUIRE_REQUEST  = "l_acq_req"
	LOCK_ACQUIRE_RESPONSE = "l_acq_resp"
	LOCK_RELEASE          = "l_rel"
	DIFF_REQUEST          = "diff_req"
	DIFF_RESPONSE         = "diff_resp"
	MALLOC_REQUEST        = "mal_req"
	FREE_REQUEST          = "free_req"
	MALLOC_REPLY          = "mal_repl"
	FREE_REPLY            = "free_repl"
	WELCOME_MESSAGE       = "WELC"
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
