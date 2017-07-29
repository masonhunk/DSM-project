package treadmarks

import (
	"DSM-project/memory"
	"DSM-project/network"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
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
	vc       *Vectorclock
	twinMap  map[int][]byte //contains twins since last sync.
	twinLock *sync.Mutex
	network.IClient
	server            network.Server
	manager           tm_Manager
	eventchanMap      map[byte]chan string
	eventNumber       byte
	eventLock         *sync.RWMutex
	lastVCFromManager *Vectorclock
	waitgroupMap      map[byte]*sync.WaitGroup
	haveLock          map[int]bool
	locks             []*sync.Mutex
	*sync.Mutex
}

func NewTreadMarks(virtualMemory memory.VirtualMemory, nrProcs, nrLocks, nrBarriers int) *TreadMarks {
	gob.RegisterName("pairs", []Pair{})
	gob.Register(Vectorclock{})
	gob.Register(TM_Message{})
	gob.Register(network.SimpleMessage{})

	tm := TreadMarks{
		VirtualMemory:     virtualMemory,
		vc:                NewVectorclock(nrProcs),
		twinMap:           make(map[int][]byte),
		nrProcs:           nrProcs,
		nrLocks:           nrLocks,
		nrBarriers:        nrBarriers,
		eventchanMap:      make(map[byte]chan string),
		waitgroupMap:      make(map[byte]*sync.WaitGroup),
		eventNumber:       byte(0),
		lastVCFromManager: NewVectorclock(nrProcs),
		eventLock:         new(sync.RWMutex),
		locks:             make([]*sync.Mutex, nrLocks),
		haveLock:          make(map[int]bool),
	}
	tm.Mutex = new(sync.Mutex)
	ds := NewDataStructure(&tm.ProcId, nrProcs)
	tm.DataStructureInterface = ds
	tm.twinLock = new(sync.Mutex)
	for i := range tm.locks {
		tm.locks[i] = new(sync.Mutex)
	}

	tm.VirtualMemory.AddFaultListener(func(addr int, faultType byte, accessType string, value byte) {
		//do fancy protocol stuff here
		//if no copy, get one.
		pageNr := tm.GetPageAddr(addr) / tm.GetPageSize()
		tm.RLockPage(pageNr)
		shouldGetCopy := !tm.HasCopy(pageNr)
		copyset := tm.GetCopyset(pageNr)
		tm.RUnlockPage(pageNr)
		if shouldGetCopy {
			tm.sendCopyRequest(pageNr, byte(copyset[len(tm.GetCopyset(pageNr))-1])) //blocks until copy has been received
		}
		//get and apply diffs before continuing
		tm.RequestAndApplyDiffs(pageNr)

		switch accessType {
		case "READ":
			tm.SetRights(addr, memory.READ_ONLY)
		case "WRITE":
			//create a twin
			tm.LockPage(pageNr)
			val := tm.PrivilegedRead(tm.GetPageAddr(addr), tm.GetPageSize())
			tm.twinLock.Lock()
			fmt.Println("We are creating a twin: ", val)
			if _, ok := tm.twinMap[pageNr]; ok {
				fmt.Println("But we already had a twin!")
			}
			tm.twinMap[pageNr] = val

			tm.PrivilegedWrite(addr, []byte{value})
			fmt.Println("After the write, the twin was ", tm.twinMap[pageNr], " and what we wrote was ", value)
			tm.twinLock.Unlock()
			tm.SetRights(addr, memory.READ_WRITE)
			tm.UnlockPage(pageNr)
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
			t.lastVCFromManager = &msg.VC
			t.incorporateIntervalsIntoDatastructures(&msg)
			t.eventchanMap[msg.Event] <- "continue"
		case DIFF_REQUEST:
			return t.Send(t.HandleDiffRequest(msg))
		case DIFF_RESPONSE:
			t.HandleDiffResponse(msg)
		case COPY_REQUEST:
			msg.Data = t.PrivilegedRead(msg.PageNr*t.GetPageSize(), t.GetPageSize())
			msg.Type = COPY_RESPONSE
			msg.From, msg.To = msg.To, msg.From
			err := t.Send(msg)
			panicOnErr(err)
		case COPY_RESPONSE:
			t.LockPage(msg.PageNr)
			fmt.Println(t.ProcId, " ---- Is handling copy response.")
			t.PrivilegedWrite(msg.PageNr*t.GetPageSize(), msg.Data)
			t.SetHasCopy(msg.PageNr, true)
			t.eventchanMap[msg.Event] <- "continue"
			t.UnlockPage(msg.PageNr)
			fmt.Println(t.ProcId, " ----- Done handling copy response.")
		default:
			return errors.New("unrecognized message type value at host: " + msg.Type)
		}
		return nil
	}
	client := network.NewClient(msgHandler)
	t.IClient = client
	for {
		if err := t.Connect(address); err != nil {
			time.Sleep(time.Millisecond * 200)
		} else {
			break
		}
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

func (t *TreadMarks) ReadInt(addr int) int {
	b, _ := t.ReadBytes(addr, 4)
	result, _ := binary.Varint(b)
	return int(result)
}

func (t *TreadMarks) WriteInt(addr int, i int) {
	buff := make([]byte, 4)
	_ = binary.PutVarint(buff, int64(i))
	t.WriteBytes(addr, buff)
}

func (t *TreadMarks) Startup() error {
	var err error
	f, err := os.Create("TreadMarksLog.csv")
	if err != nil {
		f.Close()
		log.Fatal(err)
	}
	logger := network.NewCSVStructLogger(f)
	t.server, err = network.NewServer(func(message network.Message) error { return nil }, "2000", *logger)
	if err != nil {
		logger.Close()
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
	t.vc = t.vc.Merge(message.VC)
}

func (t *TreadMarks) HandleLockAcquireRequest(msg *TM_Message) TM_Message {
	fmt.Println(t.ProcId, "  ----- Handling lock acquire request.")
	t.locks[msg.Id].Lock()
	fmt.Println(t.ProcId, "  ----- Got the lock in lock acquire request handling.")
	t.haveLock[msg.Id] = false
	t.locks[msg.Id].Unlock()
	//send write notices back and stuff
	//start by incrementing vc
	//create new interval and make write notices for all twinned pages since last sync
	fmt.Println(t.ProcId, " ---- trying to update datastructure in lock acquire request")
	t.updateDatastructures()
	fmt.Println(t.ProcId, " ---- done updating datastructure in lock acquire request")
	//find all the write notices to send
	fmt.Println(t.ProcId, " ---- trying to get unseen intervals in lock acquire request")
	msg.Intervals = t.GetAllUnseenIntervals(msg.VC)
	fmt.Println(t.ProcId, " ---- got unseen intervals in lock acquire request")
	msg.From, msg.To = msg.To, msg.From
	msg.Type = LOCK_ACQUIRE_RESPONSE
	msg.VC = *t.vc

	t.Send(msg)
	fmt.Println(t.ProcId, "Returning from handling lock acquire request.")
	return *msg
}

func (t *TreadMarks) updateDatastructures() {
	fmt.Println(t.ProcId, " --- is calling updateDatastructures")
	t.LockDatastructure()

	t.twinLock.Lock()
	if len(t.twinMap) > 0 {
		interval := t.CreateNewInterval(t.ProcId, *t.vc)
		t.LockProcArray(t.ProcId)
		for pageNr := range t.twinMap {
			t.LockPage(pageNr)
			wnrl := t.GetWritenoticeRecords(t.ProcId, pageNr)
			if len(wnrl) > 0 && wnrl[0].Diff != nil {
				wnrl[0] = t.CreateNewWritenoticeRecord(t.ProcId, pageNr, interval)
				interval.WriteNotices = append(interval.WriteNotices, wnrl[0])
				t.AddWriteNoticeRecord(t.ProcId, pageNr, wnrl[0])
			}
		}
		if len(interval.WriteNotices) > 0 {
			t.AddIntervalRecord(t.ProcId, interval)

		}
		for pageNr := range t.twinMap {
			t.UnlockPage(pageNr)
		}
		t.vc.Increment(t.ProcId)
	}
	t.twinLock.Unlock()
	t.UnlockDatastructure()
	fmt.Println(t.ProcId, " --- is returning from updateDatastructures")
}

func (t *TreadMarks) RequestAndApplyDiffs(pageNr int) {
	fmt.Println(t.ProcId, " --- Is calling RequestAndApplyDiffs for pagnr ", pageNr)
	group := new(sync.WaitGroup)
	t.RLockPage(pageNr)
	fmt.Println(t.ProcId, " ------ Got the lock for pagenr ", pageNr, " in RequestAndApplyDiffs call.")
	messages := t.GenerateDiffRequests(pageNr, group)
	fmt.Println(t.ProcId, " ----- Done generating diff requests.")
	t.RUnlockPage(pageNr)
	if len(messages) <= 0 {
		fmt.Println(t.ProcId, " ---- No messages, so just returning.")
		return
	}
	group.Add(len(messages))
	for _, msg := range messages {
		t.Send(msg)
	}

	fmt.Println(t.ProcId, " ----- is waiting for ", len(messages), " responses on diff requests.")
	group.Wait()
	fmt.Println(t.ProcId, " ----- Got all the responses.")
	//all responses have been received. Now apply them
	fmt.Println(t.ProcId, " ---- Tries to apply the new diffs")
	t.LockPage(pageNr)
	fmt.Println(t.ProcId, " ----- Got the pagelock for applying diffs.")
	di := t.NewDiffIterator(pageNr)
	fmt.Println(t.ProcId, " Created diff iterator.")
	for {
		diff := di.Next()
		if diff == nil {

			break
		}
		t.ApplyDiff(pageNr, diff)
	}
	fmt.Println(t.ProcId, " done applying, releasing lock.")
	t.UnlockPage(pageNr)
	fmt.Println(t.ProcId, " --- Is return from RequestAndApplyDiffs for pagnr ", pageNr)
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
			To:     byte(i + 1),
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
	t.LockPage(pageNr)
	mwnl := t.GetWritenoticeRecords(t.ProcId, pageNr)
	if len(mwnl) > 0 {
		if mwnl[0].Diff == nil {
			pageVal := t.PrivilegedRead(pageNr*t.GetPageSize(), t.GetPageSize())
			diff := CreateDiff(t.twinMap[pageNr], pageVal)
			mwnl[0].Diff = &diff

			fmt.Println("We created a dif, from ", t.twinMap[pageNr], " and ", pageVal, "\n"+
				"which was ", diff)

			t.twinLock.Lock()
			delete(t.twinMap, pageNr)
			t.twinLock.Unlock()
			t.SetRights(pageNr*t.GetPageSize(), memory.READ_ONLY)
		}
	}
	t.UnlockPage(pageNr)
	t.RLockPage(pageNr)
	diffDescriptions := make([]DiffDescription, 0)
	for proc := byte(1); proc <= byte(t.nrProcs); proc = proc + byte(1) {
		for _, wnr := range t.GetWritenoticeRecords(proc, pageNr) {
			if wnr.Interval.Timestamp.Compare(&vc) < 0 {
				break
			}
			if wnr.Diff != nil {
				description := DiffDescription{ProcId: proc, Timestamp: *wnr.GetTimestamp(), Diff: *wnr.Diff}
				diffDescriptions = append(diffDescriptions, description)
			}
		}
	}
	t.RUnlockPage(pageNr)
	message.To = message.From
	message.From = t.ProcId
	message.Diffs = diffDescriptions
	message.VC = *t.vc
	message.Type = DIFF_RESPONSE
	return message
}

func (t *TreadMarks) HandleDiffResponse(message TM_Message) {
	t.LockPage(message.PageNr)
	for _, diff := range message.Diffs {
		if !diff.Timestamp.IsAfter(t.vc) {
			fmt.Println(t.ProcId, " --- Tried to add diff ", diff, " while we had timestamp ", t.vc)
			fmt.Println(t.ProcId, " --- Writenotices he had for this proc was :")
			for _, wn := range t.GetWritenoticeRecords(diff.ProcId, message.PageNr) {
				fmt.Println("  ------ ", wn, " with timestamp ", wn.GetTimestamp())
			}
			t.SetDiff(message.PageNr, diff)
		} else {
			fmt.Println(t.ProcId, " --- Got diff that was newer than needed.")
		}
	}
	t.waitgroupMap[message.Event].Done()
	t.UnlockPage(message.PageNr)
}

func (t *TreadMarks) Shutdown() {
	t.Close()
	if t.ProcId == byte(1) {
		t.manager.Close()
		t.server.StopServer()
	}

}

func (t *TreadMarks) AcquireLock(id int) {
	t.locks[id].Lock()
	if t.haveLock[id] == true {
		return
	}
	c := make(chan string)
	t.eventchanMap[t.eventNumber] = c
	msg := TM_Message{
		Type:  LOCK_ACQUIRE_REQUEST,
		To:    0,
		From:  t.ProcId,
		Id:    id,
		VC:    *t.vc,
		Event: t.eventNumber,
	}
	err := t.Send(msg)
	panicOnErr(err)
	<-c

	t.eventchanMap[t.eventNumber] = nil
	t.eventNumber++
}

func (t *TreadMarks) ReleaseLock(id int) {
	t.locks[id].Unlock()
	if t.haveLock[id] {
		return
	}
	t.haveLock[id] = true
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
	//t.vc.Increment(t.ProcId)
	t.updateDatastructures()

	msg := TM_Message{
		Type:      BARRIER_REQUEST,
		To:        0,
		From:      t.ProcId,
		VC:        *t.vc,
		Id:        id,
		Event:     t.eventNumber,
		Intervals: t.GetUnseenIntervalsAtProc(t.ProcId, *t.lastVCFromManager),
	}
	err := t.Send(msg)
	panicOnErr(err)
	<-c
	t.eventchanMap[t.eventNumber] = nil
	t.eventNumber++
}

func (t *TreadMarks) incorporateIntervalsIntoDatastructures(msg *TM_Message) {
	t.LockDatastructure()
	for i := 0; i < len(msg.Intervals); i++ {
		interval := msg.Intervals[i]
		if t.GetIntervalRecord(interval.Proc, interval.Vt) != nil {
			continue
		}
		t.LockProcArray(interval.Proc)
		for _, wn := range interval.WriteNotices {
			t.LockPage(wn.PageNr)
		}
		ir := t.CreateNewInterval(interval.Proc, interval.Vt)
		for _, wn := range interval.WriteNotices {
			t.AddToCopyset(msg.From, wn.PageNr)
			if len(t.GetWritenoticeRecords(t.ProcId, wn.PageNr)) > 0 {

				if myWn := t.GetWritenoticeRecords(t.ProcId, wn.PageNr)[0]; myWn != nil {
					if myWn.Diff == nil {
						pageVal := t.PrivilegedRead(wn.PageNr*t.GetPageSize(), t.GetPageSize())
						t.twinLock.Lock()
						diff := CreateDiff(t.twinMap[wn.PageNr], pageVal)
						delete(t.twinMap, wn.PageNr)
						t.twinLock.Unlock()
						myWn.Diff = &diff
					}
				}
			}
			//prepend to write notice list and update pointers
			res := t.CreateNewWritenoticeRecord(interval.Proc, wn.PageNr, ir)
			t.AddWriteNoticeRecord(interval.Proc, wn.PageNr, res)
			//check if I have a write notice for this page with no diff at head of list. If so, create diff.

			//finally invalidate the page
			t.SetRights(wn.PageNr*t.GetPageSize(), memory.NO_ACCESS)
		}
		t.AddIntervalRecord(interval.Proc, ir)
		for _, wn := range interval.WriteNotices {
			t.UnlockPage(wn.PageNr)
		}
		t.UnlockProcArray(interval.Proc)
	}
	t.vc = t.vc.Merge(msg.VC)
	t.UnlockDatastructure()
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
