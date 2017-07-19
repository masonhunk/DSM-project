package treadmarks

import (
	"DSM-project/memory"
	"DSM-project/network"
	"errors"
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

type pageNr int

type ITreadMarks interface {
	memory.VirtualMemory
	Startup() error
	Shutdown(address string)
	AcquireLock(id int)
	ReleaseLock(id int)
	Barrier(id int)
}

type TM_Message struct {
	From      byte
	To        byte
	Type      string
	Diffs     []Diff
	Id        int
	VC        Vectorclock
	Intervals []Interval
	Event     *chan string
}

func (m TM_Message) GetFrom() byte {
	return m.From
}

func (m TM_Message) GetTo() byte {
	return m.To
}

func (m TM_Message) GetType() string {
	return m.Type
}

type TreadMarks struct {
	memory.VirtualMemory //embeds this interface type
	nrProcs              int
	procId               byte
	nrLocks              int
	nrBarriers           int
	TM_IDataStructures
	vc      Vectorclock
	copyMap map[pageNr][]byte //contains twins since last sync.
	network.IClient
}

func NewTreadMarks(virtualMemory memory.VirtualMemory, nrProcs, nrLocks, nrBarriers int) *TreadMarks {
	pageArray := make(PageArray)
	procArray := make(ProcArray, nrProcs)
	diffPool := make(DiffPool, 0)
	tm := TreadMarks{
		VirtualMemory:      virtualMemory,
		TM_IDataStructures: &TM_DataStructures{diffPool, procArray, pageArray},
		vc:                 Vectorclock{make([]uint, nrProcs)},
		copyMap:            make(map[pageNr][]byte),
		nrProcs:            nrProcs,
		nrLocks:            nrLocks,
		nrBarriers:         nrBarriers,
	}

	tm.VirtualMemory.AddFaultListener(func(addr int, faultType byte, accessType string, value byte) {
		//c := make(chan string)
		//do fancy protocol stuff here
		switch accessType {
		case "WRITE":
			//create a copy
			tm.SetRights(addr, memory.READ_WRITE)
			val, err := tm.ReadBytes(tm.GetPageAddr(addr), tm.GetPageSize())
			panicOnErr(err)
			tm.copyMap[pageNr(tm.GetPageAddr(addr)/tm.GetPageSize())] = val
			err = tm.Write(addr, value)
			panicOnErr(err)
		}
	})
	return &tm
}

func (t *TreadMarks) Startup(address string) error {
	c := make(chan bool)

	msgHandler := func(message network.Message) error {
		//handle incoming messages
		msg, ok := message.(TM_Message)
		if !ok {
			return errors.New("invalid message struct type")
		}
		switch msg.GetType() {
		case WELCOME_MESSAGE:
			t.procId = msg.To
			c <- true
		case LOCK_ACQUIRE_REQUEST:
			t.HandleLockAcquireRequest(&msg)
		case LOCK_ACQUIRE_RESPONSE:
			t.HandleLockAcquireResponse(&msg)
			*msg.Event <- "continue"
		case BARRIER_RESPONSE:
			*msg.Event <- "continue"
		case DIFF_REQUEST:

		case DIFF_RESPONSE:

		default:
			return errors.New("unrecognized message type value: " + msg.Type)
		}
		return nil
	}
	client := network.NewClient(msgHandler)
	t.IClient = client
	if err := t.Connect(address); err != nil {
		return err
	}
	<-c
	return nil
}

func (t *TreadMarks) HandleLockAcquireResponse(message *TM_Message) {
	//Here we need to add the incoming intervals to the correct write notices.
	t.incorporateIntervalsIntoDatastructures(message)
}

func (t *TreadMarks) HandleLockAcquireRequest(msg *TM_Message) TM_Message {
	//send write notices back and stuff
	//start by incrementing vc
	t.vc.Increment(t.procId)
	//create new interval and make write notices for all twinned pages since last sync
	t.updateDatastructures()
	//find all the write notices to send
	t.MapProcArray(
		func(p *Pair, procNr byte) {
			if *p != (Pair{}) && p.car != nil {
				var iRecord *IntervalRecord = p.car.(*IntervalRecord)
				//loop through the interval records for this process
				for {
					if iRecord == nil {
						break
					}
					// if this record has older ts than the requester, break
					if iRecord.Timestamp.Compare(&msg.VC) == -1 {
						break
					}
					i := Interval{
						Proc: procNr,
						Vt:   iRecord.Timestamp,
					}
					for _, wn := range iRecord.WriteNotices {
						i.WriteNotices = append(i.WriteNotices, wn.WriteNotice)
					}
					msg.Intervals = append(msg.Intervals, i)

					iRecord = iRecord.NextIr
				}
			}
		})
	msg.From, msg.To = msg.To, msg.From
	msg.Type = LOCK_ACQUIRE_RESPONSE
	msg.VC = t.vc
	return *msg
}

func (t *TreadMarks) updateDatastructures() {
	interval := IntervalRecord{
		Timestamp:    t.vc,
		WriteNotices: make([]*WriteNoticeRecord, 0),
	}
	//add interval record to front of linked list in procArray

	for key := range t.copyMap {
		//if entry doesn't exist yet, initialize it
		entry := t.GetPageEntry(int(key))
		if entry.ProcArr == nil && entry.CopySet == nil {
			t.SetPageEntry(int(key),
				PageArrayEntry{
					CopySet: []int{},
					ProcArr: make(map[byte]*WriteNoticeRecord),
				})
		}
		wn := t.PrependWriteNotice(t.procId, WriteNotice{pageNr: int(key)})
		wn.Interval = &interval
		wn.WriteNotice = WriteNotice{int(key)}
		interval.WriteNotices = append(interval.WriteNotices, wn)

	}

	//We only actually add the interval if we have any writenotices
	if len(interval.WriteNotices) > 0 {
		t.PrependIntervalRecord(t.procId, &interval)
	}
}

func (t *TreadMarks) Shutdown() {
	t.Close()
}

func (t *TreadMarks) AcquireLock(id int) {
	c := make(chan string)
	msg := TM_Message{
		Type:  LOCK_ACQUIRE_REQUEST,
		To:    1,
		From:  t.procId,
		Diffs: nil,
		Id:    id,
		VC:    t.vc,
		Event: &c,
	}
	err := t.Send(msg)
	panicOnErr(err)
	<-c
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
	c := make(chan string)
	msg := TM_Message{
		Type:  BARRIER_REQUEST,
		To:    1,
		From:  t.procId,
		Diffs: nil,
		Id:    id,
		Event: &c,
	}
	err := t.Send(msg)
	panicOnErr(err)
	<-c
}

func (t *TreadMarks) incorporateIntervalsIntoDatastructures(msg *TM_Message) {
	for i := len(msg.Intervals) - 1; i >= 0; i-- {
		interval := msg.Intervals[i]
		ir := IntervalRecord{
			Timestamp: interval.Vt,
		}
		t.PrependIntervalRecord(interval.Proc, &ir)
		for _, wn := range interval.WriteNotices {
			var res *WriteNoticeRecord = t.PrependWriteNotice(interval.Proc, wn)
			res.Interval = &ir
			ir.WriteNotices = append(ir.WriteNotices, res)
		}
	}
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
