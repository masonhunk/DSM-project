package treadmarks

import (
	"DSM-project/memory"
	"DSM-project/network"
)

const (
	LOCK_ACQUIRE_REQUEST  = "l_acq_req"
	LOCK_ACQUIRE_RESPONSE = "l_acq_resp"
	BARRIER_REQUEST       = "b_req"
	BARRIER_RESPONSE      = "b_resp"
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
	procId               byte
	nrLocks              int
	nrBarriers           int
	pageArray            PageArray
	procArray            ProcArray
	diffPool             DiffPool
	vc                   Vectorclock
	network.IClient
}

func NewTreadMarks(client network.IClient, virtualMemory memory.VirtualMemory) *TreadMarks {
	tm := TreadMarks{
		VirtualMemory: virtualMemory,
		IClient:       client,
		pageArray:     make(PageArray),
		procArray:     make(ProcArray, 0),
		diffPool:      make(DiffPool, 0),
		vc:            Vectorclock{},
	}
	tm.VirtualMemory.AddFaultListener(func(addr int, faultType byte) {
		//do fancy protocol stuff here
	})
	return &tm
}

func (t *TreadMarks) Startup() error {

}

func (t *TreadMarks) Shutdown() {
	t.Close()
}

func (t *TreadMarks) AcquireLock(id int) {
	msg := TM_Message{
		Type:  LOCK_ACQUIRE_REQUEST,
		To:    1,
		From:  t.procId,
		Diffs: nil,
		Id:    id,
		VC:    t.vc,
	}
	err := t.Send(msg)
	panicOnErr(err)
}

func (t *TreadMarks) ReleaseLock(id int) {
	msg := TM_Message{
		Type:  LOCK_RELEASE,
		To:    1,
		From:  t.procId,
		Diffs: nil,
		Id:    id,
	}
	err := t.Send(msg)
	panicOnErr(err)
}

func (t *TreadMarks) barrier(id int) {
	msg := TM_Message{
		To:    1,
		From:  t.procId,
		Diffs: nil,
		Id:    id,
	}
	err := t.Send(msg)
	panicOnErr(err)
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
