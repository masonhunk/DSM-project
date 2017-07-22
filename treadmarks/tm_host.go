package treadmarks

import (
	"DSM-project/memory"
	"DSM-project/network"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

//TODO remove when done. Is only there to make the compiler shut up.
var _ = fmt.Print

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
	COPY_REQUEST          = "copy_req"
	COPY_RESPONSE         = "copy_resp"
)

type ITreadMarks interface {
	memory.VirtualMemory
	Startup() error
	Join(address string) error
	Shutdown()
	AcquireLock(id int)
	ReleaseLock(id int)
	Barrier(id int)
}

type TM_Message struct {
	From      byte
	To        byte
	Type      string
	Diffs     []Pair
	Id        int
	PageNr    int
	VC        Vectorclock
	Intervals []Interval
	Event     *chan string
	Data      []byte
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
	ProcId               byte
	nrLocks              int
	nrBarriers           int
	TM_IDataStructures
	vc      Vectorclock
	twinMap map[int][]byte //contains twins since last sync.
	network.IClient
	server  network.Server
	manager tm_Manager
}

func NewTreadMarks(virtualMemory memory.VirtualMemory, nrProcs, nrLocks, nrBarriers int) *TreadMarks {
	gob.Register(TM_Message{})
	gob.Register(network.SimpleMessage{})

	pageArray := *NewPageArray(nrProcs)
	procArray := MakeProcArray(nrProcs)
	tm := TreadMarks{
		VirtualMemory:      virtualMemory,
		TM_IDataStructures: &TM_DataStructures{new(sync.RWMutex), procArray, pageArray},
		vc:                 *NewVectorclock(nrProcs),
		twinMap:            make(map[int][]byte),
		nrProcs:            nrProcs,
		nrLocks:            nrLocks,
		nrBarriers:         nrBarriers,
	}

	tm.VirtualMemory.AddFaultListener(func(addr int, faultType byte, accessType string, value byte) {
		//c := make(chan string)
		//do fancy protocol stuff here
		switch accessType {
		case "READ":
			pageNr := tm.GetPageAddr(addr) / tm.GetPageSize()
			//if no copy, get one. Else, create twin
			if entry := tm.GetPageEntry(pageNr); !entry.hascopy {
				tm.sendCopyRequest(pageNr, tm.ProcId)
			}
			//TODO: get and apply diffs before continuing
			tm.SetRights(addr, memory.READ_WRITE)
		case "WRITE":
			pageNr := tm.GetPageAddr(addr) / tm.GetPageSize()
			//if no copy, get one. Else create twin
			if entry := tm.GetPageEntry(pageNr); entry.GetCopyset() == nil && entry.GetWritenoticeList(tm.ProcId) == nil {
				tm.SetPageEntry(pageNr, NewPageArrayEntry(tm.nrProcs))
				tm.sendCopyRequest(pageNr, tm.ProcId)
			} else {
				//create a twin
				val := tm.PrivilegedRead(tm.GetPageAddr(addr), tm.GetPageSize())
				tm.twinMap[pageNr] = val
			}
			//TODO: get and apply diffs before continuing
			tm.SetRights(addr, memory.READ_WRITE)
			tm.PrivilegedWrite(addr, []byte{value})
		}
	})
	return &tm
}

func (t *TreadMarks) Join(address string) error {
	c := make(chan bool)

	msgHandler := func(message network.Message) error {
		log.Println("host", t.ProcId, "recieved message : ", message)

		//switch between types. If simple message, convert to tm_Message. If neither, return error
		var msg TM_Message
		if mes, ok := message.(network.SimpleMessage); ok {
			msg.To = mes.GetTo()
			msg.Type = mes.GetType()
		} else if _, ok := message.(TM_Message); !ok {
			log.Println("received invalid message struct type at host", t.ProcId)
			return errors.New("invalid message struct type")
		} else {
			msg = message.(TM_Message)
		}
		//handle incoming messages
		switch msg.GetType() {
		case WELCOME_MESSAGE:
			t.ProcId = msg.To
			c <- true
		case LOCK_ACQUIRE_REQUEST:
			t.HandleLockAcquireRequest(&msg)
		case LOCK_ACQUIRE_RESPONSE:
			t.HandleLockAcquireResponse(&msg)
			*msg.Event <- "continue"
		case BARRIER_RESPONSE:
			*msg.Event <- "continue"
		case DIFF_REQUEST:
			//remember to create own diff and set read protection on page(s)
		case DIFF_RESPONSE:

		case COPY_REQUEST:
			//if we have a twin, send that. Else just send the current contents of page
			if pg, ok := t.twinMap[msg.PageNr]; ok {
				msg.Data = pg
			} else {
				t.PrivilegedRead(msg.PageNr*t.GetPageSize(), t.GetPageSize())
				msg.Data = pg
			}
			msg.From, msg.To = msg.To, msg.From
			err := t.Send(msg)
			panicOnErr(err)
		case COPY_RESPONSE:
			t.PrivilegedWrite(msg.PageNr*t.GetPageSize(), msg.Data)
			*msg.Event <- "continue"
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
	log.Println("host successfully joined system with id:", t.ProcId)
	return nil
}

func (t *TreadMarks) Startup() error {
	var err error
	t.server, err = network.NewServer(func(message network.Message) error { return nil }, "2000")
	if err != nil {
		return err
	}
	log.Println("sucessfully started server")
	time.Sleep(time.Millisecond * 100)

	c := network.NewClient(func(message network.Message) error {
		log.Println("manager received message:", message)
		return nil
	})
	c.GetTransciever().(network.LoggingTransciever).SetLogFuncOnSend(func() {
		log.Println()
	})
	c.Connect("localhost:2000")
	bman := NewBarrierManagerImp(t.nrProcs)
	lman := NewLockManagerImp()
	t.manager = *NewTest_TM_Manager(c.GetTransciever(), bman, lman, t.nrProcs, t)
	return t.Join("localhost:2000")
}

func (t *TreadMarks) HandleLockAcquireResponse(message *TM_Message) {
	//Here we need to add the incoming intervals to the correct write notices.
	t.incorporateIntervalsIntoDatastructures(message)
	t.vc = *t.vc.Merge(message.VC)
}

func (t *TreadMarks) HandleLockAcquireRequest(msg *TM_Message) TM_Message {
	//send write notices back and stuff
	//start by incrementing vc
	t.vc.Increment(t.ProcId)
	//create new interval and make write notices for all twinned pages since last sync
	t.updateDatastructures()
	//find all the write notices to send
	msg.Intervals = t.GetAllUnseenIntervals(msg.VC)
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

	for key := range t.twinMap {
		//if entry doesn't exist yet, initialize it
		entry := t.GetPageEntry(int(key))
		if entry.GetWritenoticeList(t.ProcId) == nil && entry.GetCopyset() == nil {
			t.SetPageEntry(int(key), NewPageArrayEntry(t.nrProcs))
		}
		//add interval record to front of linked list in procArray
		wn := t.PrependWriteNotice(t.ProcId, WriteNotice{PageNr: int(key)})
		wn.Interval = &interval
		wn.WriteNotice = WriteNotice{int(key)}
		interval.WriteNotices = append(interval.WriteNotices, wn)

	}

	//We only actually add the interval if we have any writenotices
	if len(interval.WriteNotices) > 0 {
		t.PrependIntervalRecord(t.ProcId, &interval)
	}
}

func (t *TreadMarks) GenerateDiffRequests(pageNr int) []TM_Message {
	//First we check if we have the page already or need to request a copy.
	if t.twinMap[pageNr] == nil {
		//TODO We dont have a copy, so we need to request a new copy of the page.
	}
	messages := make([]TM_Message, 0)
	vc := make([]Vectorclock, t.nrProcs)
	// First we figure out what processes we actually need to send messages to.
	// To do this, we first find the largest interval for each process, where there is a write notice without
	// a diff.
	// During this, we also find the lowest timestamp for this process, where we are missing diffs.
	intrec := make([]*IntervalRecord, t.nrProcs)
	for proc := byte(0); proc < byte(t.nrProcs); proc = proc + byte(1) {
		wnr := t.TM_IDataStructures.GetWriteNoticeListHead(pageNr, proc)
		if wnr != nil && wnr.Diff == nil {
			intrec[int(proc)] = wnr.Interval
			wnrs := t.GetWritenoticeList(proc, pageNr)
			for i := 0; i < len(wnrs); i++ {
				if wnrs[i].Diff == nil {
					break
				}
				vc[int(proc)] = wnrs[i].Interval.Timestamp
			}
		}
	}
	// After that we remove the ones, that is overshadowed by others.
	for proc, int1 := range intrec {
		if int1 == nil {
			continue
		}
		overshadowed := false
		for _, int2 := range intrec {
			if int2 == nil {
				continue
			}
			if int1.Timestamp.Compare(int2.Timestamp) < 0 {
				overshadowed = true
				break
			}
		}
		if overshadowed == false {
			message := TM_Message{
				From:   t.ProcId,
				To:     byte(proc),
				Type:   DIFF_REQUEST,
				VC:     vc[proc],
				PageNr: pageNr,
			}
			messages = append(messages, message)
		}
	}
	return messages
}

func (t *TreadMarks) HandleDiffRequest(message TM_Message) TM_Message {
	//First we populate a list of pairs with all the relevant diffs.
	vc := message.VC
	pageNr := message.PageNr
	pairs := make([]Pair, 0)
	for proc := byte(0); proc < byte(t.nrProcs); proc = proc + byte(1) {
		for _, wnr := range t.GetWritenoticeList(proc, pageNr) {
			if wnr.Interval.Timestamp.Compare(vc) < 0 {
				break
			}
			if wnr.Diff != nil {
				pairs = append(pairs, Pair{wnr.Interval.Timestamp, wnr.Diff.Diffs})
			}
		}
	}
	message.To = message.From
	message.From = t.ProcId
	message.Diffs = pairs
	message.Type = DIFF_RESPONSE
	return message
}

func (t *TreadMarks) Shutdown() {
	t.Close()
	if t.manager != (tm_Manager{}) {
		t.manager.Close()
		t.server.StopServer()
	}

}

func (t *TreadMarks) AcquireLock(id int) {
	c := make(chan string)
	msg := TM_Message{
		Type:  LOCK_ACQUIRE_REQUEST,
		To:    1,
		From:  t.ProcId,
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
		From:  t.ProcId,
		Diffs: nil,
		Id:    id,
	}
	err := t.Send(msg)
	panicOnErr(err)
}

func (t *TreadMarks) Barrier(id int) {
	c := make(chan string)
	//last known timestamp from manager that this host has seen
	var managerTs Vectorclock
	if t.GetIntervalRecord(byte(0), 0) == nil {
		managerTs = *NewVectorclock(t.nrProcs)
	} else {
		managerTs = t.GetIntervalRecordHead(byte(0)).Timestamp
	}

	/*if managerTs.Compare(NewVectorclock(t.nrProcs)) == 0 {
	}*/
	msg := TM_Message{
		Type:      BARRIER_REQUEST,
		To:        0,
		From:      t.ProcId,
		Diffs:     nil,
		VC:        t.vc,
		Id:        id,
		Event:     &c,
		Intervals: t.GetUnseenIntervalsAtProc(managerTs, t.ProcId),
	}
	err := t.Send(msg)
	panicOnErr(err)
	<-c
}

func (t *TreadMarks) incorporateIntervalsIntoDatastructures(msg *TM_Message) {
	t.Lock()
	for i := len(msg.Intervals) - 1; i >= 0; i-- {
		interval := msg.Intervals[i]
		ir := &IntervalRecord{
			Timestamp:    interval.Vt,
			WriteNotices: []*WriteNoticeRecord{},
		}
		for _, wn := range interval.WriteNotices {
			//prepend to write notice list and update pointers
			var res *WriteNoticeRecord = t.PrependWriteNotice(interval.Proc, wn)
			res.Interval = ir
			ir.WriteNotices = append(ir.WriteNotices, res)
			//check if I have a write notice for this page with no diff at head of list. If so, create diff.
			if myWn := t.GetWriteNoticeListHead(wn.PageNr, t.ProcId); myWn != nil && myWn.Diff == nil {
				pageVal, err := t.ReadBytes(wn.PageNr*t.GetPageSize(), t.GetPageSize())
				panicOnErr(err)
				diff := CreateDiff(t.twinMap[wn.PageNr], pageVal)
				t.twinMap[wn.PageNr] = nil
				myWn.Diff = &diff
			}
			//finally invalidate the page.
			t.SetRights(wn.PageNr*t.GetPageSize(), memory.NO_ACCESS)
		}
		t.PrependIntervalRecord(interval.Proc, ir)

	}
	t.Unlock()
}

func (t *TreadMarks) sendCopyRequest(pageNr int, procNr byte) {
	c := make(chan string)
	msg := TM_Message{
		Type:   COPY_REQUEST,
		To:     procNr,
		From:   t.ProcId,
		Event:  &c,
		PageNr: pageNr,
	}
	err := t.Send(msg)
	panicOnErr(err)
	<-c
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
