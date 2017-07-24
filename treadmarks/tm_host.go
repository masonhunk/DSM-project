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
	Diffs     []DiffDescription
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
	DataStructureInterface
	vc      Vectorclock
	twinMap map[int][]byte //contains twins since last sync.
	network.IClient
	server            network.Server
	manager           tm_Manager
	eventchanMap      map[byte]chan string
	eventNumber       byte
	eventLock         *sync.RWMutex
	lastVCFromManager Vectorclock
	waitgroupMap      map[byte]*sync.WaitGroup
}

func NewTreadMarks(virtualMemory memory.VirtualMemory, nrProcs, nrLocks, nrBarriers int) *TreadMarks {
	gob.RegisterName("pairs", []Pair{})
	gob.Register(Vectorclock{})
	gob.Register(TM_Message{})
	gob.Register(network.SimpleMessage{})

	tm := TreadMarks{
		VirtualMemory:     virtualMemory,
		vc:                *NewVectorclock(nrProcs),
		twinMap:           make(map[int][]byte),
		nrProcs:           nrProcs,
		nrLocks:           nrLocks,
		nrBarriers:        nrBarriers,
		eventchanMap:      make(map[byte]chan string),
		waitgroupMap:      make(map[byte]*sync.WaitGroup),
		eventNumber:       byte(0),
		lastVCFromManager: *NewVectorclock(nrProcs),
		eventLock:         new(sync.RWMutex),
	}
	ds := NewDataStructure(&tm.ProcId, nrProcs)
	tm.DataStructureInterface = *ds

	tm.VirtualMemory.AddFaultListener(func(addr int, faultType byte, accessType string, value byte) {
		//do fancy protocol stuff here
		//if no copy, get one.
		pageNr := tm.GetPageAddr(addr) / tm.GetPageSize()
		if !tm.HasCopy(pageNr) {
			copyset := tm.GetCopyset(pageNr)
			tm.sendCopyRequest(pageNr, byte(copyset[len(tm.GetCopyset(pageNr))-1])) //blocks until copy has been received
			tm.SetHasCopy(pageNr, true)
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
			//if we have a twin, send that. Else just send the current contents of page
			if pg, ok := t.twinMap[msg.PageNr]; ok {
				msg.Data = pg
			} else {
				t.PrivilegedRead(msg.PageNr*t.GetPageSize(), t.GetPageSize())
				msg.Data = pg
			}
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
	interval := t.CreateNewInterval(t.ProcId, t.vc)

	for key := range t.twinMap {
		wn := t.CreateNewWritenoticeRecord(t.ProcId, key, interval)
		t.AddWriteNoticeRecord(t.ProcId, key, wn)

	}

	//We only actually add the interval if we have any writenotices
	if len(interval.WriteNotices) > 0 {
		t.AddIntervalRecord(t.ProcId, interval)
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

	di := t.NewDiffIterator(pageNr)
	for {
		diff := di.Next()
		fmt.Println("got this diff:", diff)
		if diff != nil {
			t.ApplyDiff(pageNr, diff)
		}
	}
}

func (t *TreadMarks) GenerateDiffRequests(pageNr int, group *sync.WaitGroup) []TM_Message {
	//First we check if we have the page already or need to request a copy.
	pairs := t.GetMissingDiffTimestamps(pageNr)
	//Then we build the messages
	messages := make([]TM_Message, 0)

	for i := 0; i < t.nrProcs; i++ {
		if pairs[i].Car == nil || pairs[i].Cdr == nil {
			continue
		}
		endTs := pairs[i].Cdr.(Vectorclock)

		message := TM_Message{
			From:   t.ProcId,
			To:     byte(i),
			Type:   DIFF_REQUEST,
			VC:     endTs,
			PageNr: pageNr,
			Event:  t.eventNumber,
		}
		messages = append(messages, message)
	}
	if len(messages) > 0 {
		t.eventLock.Lock()
		t.waitgroupMap[t.eventNumber] = group
		t.eventNumber++
		t.eventLock.Unlock()
	}
	return messages
}

func (t *TreadMarks) HandleDiffRequest(message TM_Message) TM_Message {

	//First we populate a list of pairs with all the relevant diffs.
	vc := message.VC
	pageNr := message.PageNr
	mwnl := t.GetWritenoticeRecords(t.ProcId, pageNr)
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
	diffDescriptions := make([]DiffDescription, 0)
	for proc := byte(0); proc < byte(t.nrProcs); proc = proc + byte(1) {
		for _, wnr := range t.GetWritenoticeRecords(proc, pageNr) {
			if wnr.Interval.Timestamp.Compare(vc) < 0 {
				break
			}
			if wnr.Diff != nil {
				description := DiffDescription{Timestamp: *wnr.GetTimestamp(), Diff: *wnr.Diff}
				diffDescriptions = append(diffDescriptions, description)
			}
		}
	}
	message.To = message.From
	message.From = t.ProcId
	message.Diffs = diffDescriptions
	message.VC = t.vc
	message.Type = DIFF_RESPONSE
	return message
}

func (t *TreadMarks) HandleDiffResponse(message TM_Message) {

	t.SetDiffs(message.PageNr, message.Diffs)
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
	fmt.Println("Working at proc ", t.ProcId)
	fmt.Println("Which has VC :", t.vc)
	t.vc.Increment(t.ProcId)
	t.updateDatastructures()
	msg := TM_Message{
		Type:      BARRIER_REQUEST,
		To:        0,
		From:      t.ProcId,
		VC:        t.vc,
		Id:        id,
		Event:     t.eventNumber,
		Intervals: t.GetUnseenIntervalsAtProc(t.ProcId, t.lastVCFromManager),
	}
	fmt.Println("Sending barrier request with vectorclock ", msg.VC)
	err := t.Send(msg)
	panicOnErr(err)
	<-c
	t.eventchanMap[t.eventNumber] = nil
	t.eventNumber++
}

func (t *TreadMarks) incorporateIntervalsIntoDatastructures(msg *TM_Message) {

	for i := len(msg.Intervals) - 1; i >= 0; i-- {
		interval := msg.Intervals[i]
		ir := t.CreateNewInterval(interval.Proc, interval.Vt)

		for _, wn := range interval.WriteNotices {

			t.AddToCopyset(msg.From, wn.PageNr)
			//prepend to write notice list and update pointers
			res := t.CreateNewWritenoticeRecord(interval.Proc, wn.PageNr, ir)
			t.AddWriteNoticeRecord(interval.Proc, wn.PageNr, res)

			//check if I have a write notice for this page with no diff at head of list. If so, create diff.
			if myWn := t.GetWritenoticeRecords(t.ProcId, wn.PageNr)[0]; myWn != nil {
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
		t.AddIntervalRecord(interval.Proc, ir)
	}
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
