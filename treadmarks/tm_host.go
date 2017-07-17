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
	Barrier(id int)
}

type TM_Message struct {
	From         byte
	To           byte
	Type         string
	Diffs        []Diff
	Id           int
	VC           Vectorclock
	WriteNotices []WriteNotice
}

func (m *TM_Message) GetFrom() byte {
	return m.From
}

func (m *TM_Message) GetTo() byte {
	return m.To
}

func (m *TM_Message) GetType() string {
	return m.Type
}

type TreadMarks struct {
	memory.VirtualMemory //embeds this interface type
	nrProcs              int
	procId               int
	nrLocks              int
	nrBarriers           int
	pageArray            PageArray
	procArray            ProcArray
	diffPool             DiffPool
	network.IClient
}

func NewTreadMarks(client network.IClient, virtualMemory memory.VirtualMemory) *TreadMarks {
	tm := TreadMarks{
		VirtualMemory: virtualMemory,
	}
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
