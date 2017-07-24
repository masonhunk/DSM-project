package treadmarks

import (
	"DSM-project/memory"
	"DSM-project/network"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
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
	Event     byte
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
	memory.VirtualMemory     //embeds this interface type
	nrProcs              int //including the manager process
	ProcId               byte
	nrLocks              int
	nrBarriers           int
	TM_IDataStructures
	vc      Vectorclock
	twinMap map[int][]byte //contains twins since last sync.
	network.IClient
	server            network.Server
	manager           tm_Manager
	eventchanMap      map[byte]chan string
	eventNumber       byte
	lastVCFromManager Vectorclock
	waitgroupMap      map[byte]*sync.WaitGroup
}

func NewTreadMarks(virtualMemory memory.VirtualMemory, nrProcs, nrLocks, nrBarriers int) *TreadMarks {
	gob.RegisterName("pairs", []Pair{})
	gob.Register(Vectorclock{})
	gob.Register(TM_Message{})
	gob.Register(network.SimpleMessage{})

	pageArray := *NewPageArray(nrProcs)
	procArray := MakeProcArray(nrProcs + 1)
	tm := TreadMarks{
		VirtualMemory:      virtualMemory,
		TM_IDataStructures: &TM_DataStructures{new(sync.RWMutex), procArray, pageArray},
		vc:                 *NewVectorclock(nrProcs + 1),
		twinMap:            make(map[int][]byte),
		nrProcs:            nrProcs,
		nrLocks:            nrLocks,
		nrBarriers:         nrBarriers,
		eventchanMap:       make(map[byte]chan string),
		waitgroupMap:       make(map[byte]*sync.WaitGroup),
		eventNumber:        byte(0),
		lastVCFromManager:  *NewVectorclock(nrProcs + 1),
	}

	tm.VirtualMemory.AddFaultListener(func(addr int, faultType byte, accessType string, value byte) {
		//do fancy protocol stuff here
		//if no copy, get one.
		pageNr := tm.GetPageAddr(addr) / tm.GetPageSize()
		if entry := tm.GetPageEntry(pageNr); !entry.HasCopy() {
			copyset := tm.GetCopyset(pageNr)
			tm.sendCopyRequest(pageNr, byte(copyset[len(tm.GetCopyset(pageNr))-1])) //blocks until copy has been received
			entry.SetHasCopy(true)
		}
		//get and apply diffs before continuing
		tm.RequestAndApplyDiffs(pageNr)

		switch accessType {
		case "READ":
			tm.SetRights(addr, memory.READ_ONLY)
		case "WRITE":
			//create a twin
			val := tm.PrivilegedRead(tm.GetPageAddr(addr), tm.GetPageSize())
			tm.twinMap[pageNr] = val
			tm.PrivilegedWrite(addr, []byte{value})
			tm.SetRights(addr, memory.READ_WRITE)
		}
	})
	return &tm
}

func (t *TreadMarks) Join(address string) error {
	c := make(chan bool)

	msgHandler := func(message network.Message) error {

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
			t.eventchanMap[msg.Event] <- "continue"
		case BARRIER_RESPONSE:
			t.lastVCFromManager = msg.VC
			t.vc = msg.VC
			t.incorporateIntervalsIntoDatastructures(&msg)
			t.eventchanMap[msg.Event] <- "continue"
		case DIFF_REQUEST:
			return t.Send(t.HandleDiffRequest(msg))
		case DIFF_RESPONSE:
			t.HandleDiffResponse(msg)
		case COPY_REQUEST:
			//Just send current contents
			msg.Data = t.PrivilegedRead(msg.PageNr*t.GetPageSize(), t.GetPageSize())
			/*
				//if we have a twin, send that. Else just send the current contents of page
				if pg, ok := t.twinMap[msg.PageNr]; ok {
					msg.Data = pg
				} else {
					pg := t.PrivilegedRead(msg.PageNr*t.GetPageSize(), t.GetPageSize())
					msg.Data = pg
				}*/
			msg.Type = COPY_RESPONSE
			msg.From, msg.To = msg.To, msg.From
			err := t.Send(msg)
			panicOnErr(err)
		case COPY_RESPONSE:
			t.PrivilegedWrite(msg.PageNr*t.GetPageSize(), msg.Data)
			t.eventchanMap[msg.Event] <- "continue"
		default:
			return errors.New("unrecognized message type value at host: " + msg.Type)
		}
		return nil
	}
	client := network.NewClient(msgHandler)
	t.IClient = client
	if err := t.Connect(address); err != nil {
		return err
	}
	<-c
	client.GetTransciever().(network.LoggableTransciever).SetLogFuncOnSend(func(message network.Message) {
		log.Printf("%s%d%s%+v", "host", t.ProcId, " sent message:", message)
	})
	client.GetTransciever().(network.LoggableTransciever).SetLogFuncOnReceive(func(message network.Message) {
		log.Printf("%s%d%s%+v", "host", t.ProcId, " received message:", message)
	})
	client.GetTransciever().(network.LoggableTransciever).ShouldLog(true)

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

	conn, err := net.Dial("tcp", "localhost:2000")
	bman := NewBarrierManagerImp(t.nrProcs)
	lman := NewLockManagerImp()
	t.manager = *NewTM_Manager(conn, bman, lman, t)
	t.manager.ITransciever.(network.LoggableTransciever).SetLogFuncOnSend(func(message network.Message) {
		log.Printf("%s%+v", "manager sent message:", message)
	})
	t.manager.ITransciever.(network.LoggableTransciever).SetLogFuncOnReceive(func(message network.Message) {
		log.Printf("%s%+v", "manager received message:", message)
	})
	t.manager.ITransciever.(network.LoggableTransciever).ShouldLog(true)
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
	t.Send(msg)
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

func (t *TreadMarks) RequestAndApplyDiffs(pageNr int) {
	group := new(sync.WaitGroup)
	messages := t.GenerateDiffRequests(pageNr, group)
	if len(messages) <= 0 {
		return
	}
	group.Add(len(messages))
	for _, msg := range messages {
		t.Send(msg)
	}
	group.Wait()
	//all responses have been received. Now apply them
	pe := t.GetPageEntry(pageNr)
	channel := pe.OrderedDiffChannel()
	//fmt.Println("write notices at page 1 from proc 2:", t.GetWritenoticeList(byte(2), 1))
	for diff := range channel {
		fmt.Println("got this diff:", diff)
		if diff != nil {
			t.ApplyDiff(pageNr, diff)
		}
	}
}

func (t *TreadMarks) GenerateDiffRequests(pageNr int, group *sync.WaitGroup) []TM_Message {
	//First we check if we have the page already or need to request a copy.
	if t.GetPageEntry(pageNr).hascopy == false {
		//t.sendCopyRequest(pageNr, byte(t.GetCopyset(pageNr)[0]))
	}
	//First we find the start timestamps
	ProcStartTS := make([]Vectorclock, t.nrProcs+1)
	ProcEndTS := make([]Vectorclock, t.nrProcs+1)
	for i := 0; i < t.nrProcs+1; i++ {
		wnrl := t.GetWritenoticeList(byte(i), pageNr)
		if len(wnrl) < 1 {
			continue
		}
		ProcStartTS[i] = wnrl[0].Interval.Timestamp
		for _, wnr := range wnrl {
			if wnr.Diff != nil {
				break
			}
			ProcEndTS[i] = wnr.Interval.Timestamp
		}
	}

	//Then we "merge" the different intervals
	for i := 0; i < t.nrProcs+1; i++ {
		if ProcStartTS[i].Value == nil || i == int(t.ProcId) {
			continue
		}
		for j := i; j < t.nrProcs+1; j++ {
			if ProcStartTS[j].Value == nil || j == int(t.ProcId) {
				continue
			}
			if ProcStartTS[i].IsAfter(ProcStartTS[j]) {
				if ProcEndTS[i].Value == nil || ProcEndTS[i].IsAfter(ProcEndTS[j]) {
					ProcEndTS[i] = ProcEndTS[j]
				}
				ProcEndTS[j] = Vectorclock{}
			} else if ProcStartTS[j].IsAfter(ProcStartTS[i]) {
				if ProcEndTS[j].Value == nil || ProcEndTS[j].IsAfter(ProcEndTS[i]) {
					ProcEndTS[j] = ProcEndTS[i]
				}
				ProcEndTS[i] = Vectorclock{}
			}
		}

	}

	//Then we build the messages
	messages := make([]TM_Message, 0)

	for i := 0; i < t.nrProcs+1; i++ {
		if ProcStartTS[i].Value == nil || ProcEndTS[i].Value == nil {
			continue
		}
		message := TM_Message{
			From:   t.ProcId,
			To:     byte(i),
			Type:   DIFF_REQUEST,
			VC:     ProcEndTS[i],
			PageNr: pageNr,
			Event:  t.eventNumber,
		}
		messages = append(messages, message)
	}
	if len(messages) > 0 {
		t.waitgroupMap[t.eventNumber] = group
		t.eventNumber++
	}
	return messages
}

func (t *TreadMarks) HandleDiffRequest(message TM_Message) TM_Message {

	//First we populate a list of pairs with all the relevant diffs.
	vc := message.VC
	pageNr := message.PageNr
	mwnl := t.GetWritenoticeList(t.ProcId, pageNr)
	if len(mwnl) > 0 {
		if mwnl[0].Diff == nil {
			pageVal, err := t.ReadBytes(pageNr*t.GetPageSize(), t.GetPageSize())
			panicOnErr(err)
			diff := CreateDiff(t.twinMap[pageNr], pageVal)
			mwnl[0].Diff = &diff
			t.twinMap[pageNr] = nil
			t.SetRights(pageNr*t.GetPageSize(), memory.READ_ONLY)
		}
	}
	pairs := make([]Pair, 0)
	for proc := byte(0); proc < byte(t.nrProcs+1); proc = proc + byte(1) {
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
	message.VC = t.vc
	message.Type = DIFF_RESPONSE
	return message
}

func (t *TreadMarks) HandleDiffResponse(message TM_Message) {
	pairs := message.Diffs
	i := 0 //diff nr in msg
	j := 0 //index of process in page array entry
	for {
		if j > t.nrProcs {
			break
		}
		wnl := t.GetWritenoticeList(byte(j), message.PageNr)
		for k, wn := range wnl {
			if i >= len(pairs) {
				t.waitgroupMap[message.Event].Done()
				return
			} else if wn.Interval.Timestamp.Equals(pairs[i].Car.(Vectorclock)) {
				diff := new(Diff)
				diff.Diffs = pairs[i].Cdr.([]Pair)
				fmt.Println("saw diff:", diff)
				//diff.Diffs = pairs[i].Cdr.(Diff).Diffs
				wnl[k].Diff = diff
				fmt.Println("write notice:", t.GetWritenoticeList(byte(j), message.PageNr)[k].Diff)
				i++
			} else if wn.Interval.Timestamp.IsBefore(pairs[i].Car.(Vectorclock)) {
				break
			}
		}
		j++
	}
	t.waitgroupMap[message.Event].Done()
}

func (t *TreadMarks) Shutdown() {
	t.Close()
	if t.ProcId == byte(1) {
		t.manager.Close()
		t.server.StopServer()
	}

}

func (t *TreadMarks) AcquireLock(id int) {
	c := make(chan string)
	t.eventchanMap[t.eventNumber] = c
	msg := TM_Message{
		Type:  LOCK_ACQUIRE_REQUEST,
		To:    0,
		From:  t.ProcId,
		Id:    id,
		VC:    t.vc,
		Event: t.eventNumber,
	}
	err := t.Send(msg)
	panicOnErr(err)
	<-c
	t.eventchanMap[t.eventNumber] = nil
	t.eventNumber++
}

func (t *TreadMarks) ReleaseLock(id int) {
	msg := TM_Message{
		Type: LOCK_RELEASE,
		To:   0,
		From: t.ProcId,
		Id:   id,
	}
	err := t.Send(msg)
	panicOnErr(err)
}

func (t *TreadMarks) Barrier(id int) {
	c := make(chan string)
	t.eventchanMap[t.eventNumber] = c
	t.vc.Increment(t.ProcId)
	t.updateDatastructures()
	msg := TM_Message{
		Type:      BARRIER_REQUEST,
		To:        0,
		From:      t.ProcId,
		VC:        t.vc,
		Id:        id,
		Event:     t.eventNumber,
		Intervals: t.GetUnseenIntervalsAtProc(t.lastVCFromManager, t.ProcId),
	}
	err := t.Send(msg)
	panicOnErr(err)
	<-c
	t.eventchanMap[t.eventNumber] = nil
	t.eventNumber++
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

			//add sender to copyset of this page
			if t.GetPageEntry(wn.PageNr) == nil {
				log.Println("nil!")
			}
			t.SetCopyset(wn.PageNr, []int{int(msg.From)})
			//prepend to write notice list and update pointers
			var res *WriteNoticeRecord = t.PrependWriteNotice(interval.Proc, wn)
			res.Interval = ir
			ir.WriteNotices = append(ir.WriteNotices, res)

			//check if I have a write notice for this page with no diff at head of list. If so, create diff.
			if myWn := t.GetWriteNoticeListHead(wn.PageNr, t.ProcId); myWn != nil {
				if myWn.Diff == nil {
					pageVal := t.PrivilegedRead(wn.PageNr*t.GetPageSize(), t.GetPageSize())
					diff := CreateDiff(t.twinMap[wn.PageNr], pageVal)
					t.twinMap[wn.PageNr] = nil
					myWn.Diff = &diff
				}
			}

			//finally invalidate the page
			t.SetRights(wn.PageNr*t.GetPageSize(), memory.NO_ACCESS)
		}
		t.PrependIntervalRecord(interval.Proc, ir)

	}
	t.Unlock()
}

func (t *TreadMarks) sendCopyRequest(pageNr int, procNr byte) {
	c := make(chan string)
	t.eventchanMap[t.eventNumber] = c

	msg := TM_Message{
		Type:   COPY_REQUEST,
		To:     procNr,
		From:   t.ProcId,
		Event:  t.eventNumber,
		PageNr: pageNr,
	}
	err := t.Send(msg)
	panicOnErr(err)
	<-c
	t.eventchanMap[t.eventNumber] = nil
	t.eventNumber++
}

func (t *TreadMarks) ApplyDiff(pageNr int, diffList *Diff) {
	addr := pageNr * t.GetPageSize()
	data := t.PrivilegedRead(addr, t.GetPageSize())
	for _, diff := range diffList.Diffs {
		data[diff.Car.(int)] = diff.Cdr.(byte)
	}
	t.PrivilegedWrite(addr, data)
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
