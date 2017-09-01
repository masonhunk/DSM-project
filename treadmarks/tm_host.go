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
/*
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
			tm.twinLock.Lock()
			tm.LockPage(pageNr)
			val := tm.PrivilegedRead(tm.GetPageAddr(addr), tm.GetPageSize())

			fmt.Println(tm.ProcId, "We are creating a twin: ", val)
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
	})*/
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
			t.PrivilegedWrite(msg.PageNr*t.GetPageSize(), msg.Data)
			t.SetHasCopy(msg.PageNr, true)
			t.eventchanMap[msg.Event] <- "continue"
			t.UnlockPage(msg.PageNr)
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
	t.server, err = network.NewServer(func(message network.Message) error { return nil }, "2000", logger)
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
	fmt.Println(t.ProcId, " --- Got lock acquire response: ", message)
	//Here we need to add the incoming intervals to the correct write notices.
	t.incorporateIntervalsIntoDatastructures(message)
	t.vc = t.vc.Merge(message.VC)
}

func (t *TreadMarks) HandleLockAcquireRequest(msg *TM_Message) TM_Message {
	fmt.Println(t.ProcId, " ---- Got Lock acquire request: ", msg)
	t.locks[msg.Id].Lock()
	t.haveLock[msg.Id] = false
	t.locks[msg.Id].Unlock()
	//send write notices back and stuff
	//start by incrementing vc
	//create new interval and make write notices for all twinned pages since last sync
	t.updateDatastructures()
	//find all the write notices to send
	msg.Intervals = t.GetAllUnseenIntervals(msg.VC)
	msg.From, msg.To = msg.To, msg.From
	msg.Type = LOCK_ACQUIRE_RESPONSE
	msg.VC = *t.vc
	fmt.Println(t.ProcId, " ---- Sent lock acquire response: ", msg)
	t.Send(msg)
	return *msg
}

func (t *TreadMarks) updateDatastructures() {
	t.LockDatastructure()
	fmt.Println(t.ProcId, " --- Updating datastructure.")
	t.twinLock.Lock()
	fmt.Println(t.ProcId, " ---- Had the following twins:")
	for k, v := range t.twinMap {
		fmt.Println(t.ProcId, " ---- key: ", k, " value: ", v)
	}
	if len(t.twinMap) > 0 {
		interval := t.CreateNewInterval(t.ProcId, *t.vc)
		t.LockProcArray(t.ProcId)
		defer t.UnlockProcArray(t.ProcId)
		for pageNr := range t.twinMap {
			fmt.Println(t.ProcId, " ----- Looking at twin for page ", pageNr)
			t.LockPage(pageNr)
			defer t.UnlockPage(pageNr)
			wnrl := t.GetWritenoticeRecords(t.ProcId, pageNr)
			fmt.Println(t.ProcId, " ---- We had the following writenotices:")
			fmt.Println(t.ProcId, wnrl)
			if (len(wnrl) > 0 && wnrl[0].Diff != nil) || len(wnrl) == 0 {
				fmt.Println(t.ProcId, " ---- Writenotice for this page didnt have a diff, so we shouldnt add a new writenotice.")
				wnr := t.CreateNewWritenoticeRecord(t.ProcId, pageNr, interval)
				t.AddWriteNoticeRecord(t.ProcId, pageNr, wnr)
				fmt.Println(t.ProcId, " ---- After adding we had the following writenotices: ", t.GetWritenoticeRecords(t.ProcId, pageNr))
			} else {
				fmt.Println(t.ProcId, " ----- Either we had no writenotices, or we had already made writenotices.")
			}
		}
		if len(interval.WriteNotices) > 0 {
			t.AddIntervalRecord(t.ProcId, interval)

		}
		t.vc.Increment(t.ProcId)
	}
	t.twinLock.Unlock()
	t.UnlockDatastructure()
}

func (t *TreadMarks) RequestAndApplyDiffs(pageNr int) {
	group := new(sync.WaitGroup)
	t.RLockPage(pageNr)
	messages := t.GenerateDiffRequests(pageNr, group)
	t.RUnlockPage(pageNr)
	if len(messages) <= 0 {
		return
	}
	group.Add(len(messages))
	for _, msg := range messages {
		t.Send(msg)
	}

	group.Wait()
	//all responses have been received. Now apply them
	t.LockPage(pageNr)
	di := t.NewDiffIterator(pageNr)
	fmt.Println(t.ProcId, " ---------- Applying diffs")
	for {
		diff := di.Next()
		fmt.Println(t.ProcId, " ----------- First dif was ", diff)
		if diff == nil {
			break
		}
		fmt.Println(t.ProcId, " ------------ Data before diff applied: ", t.VirtualMemory.PrivilegedRead(pageNr*t.GetPageSize(), t.GetPageSize()))
		t.ApplyDiff(pageNr, diff)
		fmt.Println(t.ProcId, " ------------ Data after diff applied: ", t.VirtualMemory.PrivilegedRead(pageNr*t.GetPageSize(), t.GetPageSize()))
	}
	t.UnlockPage(pageNr)
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
	fmt.Println(t.ProcId, " ---- Got diff request: ", message)
	//First we populate a list of pairs with all the relevant diffs.
	vc := message.VC
	pageNr := message.PageNr
	t.twinLock.Lock()
	t.LockPage(pageNr)
	mwnl := t.GetWritenoticeRecords(t.ProcId, pageNr)
	if len(mwnl) > 0 {
		if mwnl[0].Diff == nil {
			pageVal := t.PrivilegedRead(pageNr*t.GetPageSize(), t.GetPageSize())
			diff := CreateDiff(t.twinMap[pageNr], pageVal)
			mwnl[0].Diff = &diff

			fmt.Println("We created a dif, from ", t.twinMap[pageNr], " and ", pageVal, "\n"+
				"which was ", diff)

			fmt.Println("Deleting twin ", pageNr)
			delete(t.twinMap, pageNr)
			fmt.Println("Deleted twin ", pageNr)

			t.SetRights(pageNr*t.GetPageSize(), memory.READ_ONLY)
		}
	}
	t.twinLock.Unlock()
	t.UnlockPage(pageNr)
	t.RLockPage(pageNr)
	diffDescriptions := make([]DiffDescription, 0)
	for proc := byte(1); proc <= byte(t.nrProcs); proc = proc + byte(1) {
		for _, wnr := range t.GetWritenoticeRecords(proc, pageNr) {
			if wnr.Interval.Timestamp.Compare(vc) < 0 {
				break
			}
			if wnr.Diff != nil {

				description := DiffDescription{ProcId: proc, Timestamp: wnr.GetTimestamp(), Diff: *wnr.Diff}
				fmt.Println(t.ProcId, " ---- Adding diff description: ", description)
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
	fmt.Println(t.ProcId, " ---- Answered diff request with ", message)
	return message
}

func (t *TreadMarks) HandleDiffResponse(message TM_Message) {
	fmt.Println(t.ProcId, " ---- Got diff response: ", message)
	t.LockPage(message.PageNr)
	for _, diff := range message.Diffs {
		if !diff.Timestamp.IsAfter(*t.vc) {
			t.SetDiff(message.PageNr, diff)
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
	fmt.Println(t.ProcId, " ----- Trying to incorporate intervals into datastructure.")
	fmt.Println(t.ProcId, " ----- Trying to lock datastructure.")
	t.LockDatastructure()
	fmt.Println(t.ProcId, " ----- Locked datastructure.")
	t.twinLock.Lock()
	for i := 0; i < len(msg.Intervals); i++ {

		interval := msg.Intervals[i]
		fmt.Println(t.ProcId, " ---- Incoroporating interval: ", interval)
		if t.GetIntervalRecord(interval.Proc, interval.Vt) != nil {
			fmt.Println(t.ProcId, " ----- Already had interval record.")
			continue
		}
		fmt.Println(t.ProcId, " ---- Trying to lock proc array")
		t.LockProcArray(interval.Proc)
		fmt.Println(t.ProcId, " ---- Locked proc array")
		for _, wn := range interval.WriteNotices {
			fmt.Println(t.ProcId, " ---- Trying to lock page ", wn.PageNr)
			t.LockPage(wn.PageNr)

			fmt.Println(t.ProcId, " ---- Locked page ", wn.PageNr)
		}
		fmt.Println(t.ProcId, " ---- Creating a new interval.")
		ir := t.CreateNewInterval(interval.Proc, interval.Vt)
		for _, wn := range interval.WriteNotices {
			fmt.Println(t.ProcId, " ----- adding writenotice ", wn)
			t.AddToCopyset(msg.From, wn.PageNr)
			if len(t.GetWritenoticeRecords(t.ProcId, wn.PageNr)) > 0 {
				fmt.Println(t.ProcId, " ----- We already had some writenotices for this page.")
				if myWn := t.GetWritenoticeRecords(t.ProcId, wn.PageNr)[0]; myWn != nil {
					fmt.Println(t.ProcId, " ----- We had a writenotice for this page, which we made ourselves.")
					if myWn.Diff == nil {
						fmt.Println(t.ProcId, " ---- Our writenotice didnt have a diff, so we make the diff.")
						pageVal := t.PrivilegedRead(wn.PageNr*t.GetPageSize(), t.GetPageSize())
						fmt.Println(t.ProcId, " ----- Locking twinMap")

						fmt.Println(t.ProcId, " ---- Making diff and applying it to our writenotice.")
						diff := CreateDiff(t.twinMap[wn.PageNr], pageVal)
						delete(t.twinMap, wn.PageNr)
						fmt.Println(t.ProcId, " ---- Unlocking twinmap.")

						myWn.Diff = &diff
					}
				}
			}

			fmt.Println(t.ProcId, " ----- Creating and adding the writenotice we got from the interval.")
			//prepend to write notice list and update pointers
			res := t.CreateNewWritenoticeRecord(interval.Proc, wn.PageNr, ir)
			t.AddWriteNoticeRecord(interval.Proc, wn.PageNr, res)
			//check if I have a write notice for this page with no diff at head of list. If so, create diff.
			fmt.Println(t.ProcId, " ---- Setting correct access rights.")
			//finally invalidate the page
			t.SetRights(wn.PageNr*t.GetPageSize(), memory.NO_ACCESS)
		}
		for _, wn := range interval.WriteNotices {
			fmt.Println(t.ProcId, " ---- Trying to Unlock page ", wn.PageNr)
			t.UnlockPage(wn.PageNr)

			fmt.Println(t.ProcId, " ---- Unlocked page ", wn.PageNr)
		}
		fmt.Println(t.ProcId, " ----- Adding interval record")
		t.AddIntervalRecord(interval.Proc, ir)
		fmt.Println(t.ProcId, " ----- Unlocking proc array")
		t.UnlockProcArray(interval.Proc)
	}
	fmt.Println(t.ProcId, " ---- updating vectorclock")
	t.vc = t.vc.Merge(msg.VC)
	fmt.Println(t.ProcId, " ---- Unlocking datastructure.")
	t.UnlockDatastructure()
	t.twinLock.Unlock()
	fmt.Println(t.ProcId, " ----- Done incorporating into datastructure.")
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
