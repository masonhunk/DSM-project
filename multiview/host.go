package multiview

import (
	"DSM-project/memory"
	"DSM-project/network"
	"DSM-project/treadmarks"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	READ_REQUEST          = "RR"
	WRITE_REQUEST         = "WR"
	READ_REPLY            = "RRPL"
	WRITE_REPLY           = "WRPL"
	INVALIDATE_REPLY      = "INV_REPL"
	INVALIDATE_REQUEST    = "INV_REQ"
	MALLOC_REQUEST        = "MR"
	FREE_REQUEST          = "FR"
	MALLOC_REPLY          = "MRPL"
	FREE_REPLY            = "FRPL"
	WELCOME_MESSAGE       = "WELC"
	READ_ACK              = "RA"
	WRITE_ACK             = "WA"
	LOCK_ACQUIRE_REQUEST  = "lock_acq_req"
	LOCK_ACQUIRE_RESPONSE = "lock_acq_resp"
	LOCK_RELEASE          = "lock_rel"
	BARRIER_REQUEST       = "barr_req"
	BARRIER_RESPONSE      = "barr_resp"
	MULTI_MALLOC_REQUEST  = "MMR"
	MULTI_MALLOC_REPLY    = "MMRPL"
)

type Multiview struct {
	conn             network.IClient
	mem              *hostMem
	Id               byte
	chanMap          map[int]chan string
	sequenceNumber   int
	hasLock          map[int]bool
	csvLogger        *network.CSVStructLogger
	shouldLogNetwork bool
	messagesSent     []int
}

type hostMem struct {
	vm        memory.VirtualMemory
	accessMap []byte
	//accessMap      map[int]byte //key = vpage number, value, access right
	faultListeners []memory.FaultListener
	*sync.RWMutex
}

func (m *Multiview) getInAccessMap(vpageNr int) byte {
	res := m.mem.accessMap[vpageNr]
	return res
}

func (m *Multiview) setInAccessMap(vpageNr int, val byte) {
	m.mem.Lock()
	m.mem.accessMap[vpageNr] = val
	m.mem.Unlock()
}

func NewMultiView() *Multiview {
	m := new(Multiview)
	m.sequenceNumber = 0
	m.chanMap = make(map[int]chan string)
	m.hasLock = make(map[int]bool)
	return m
}

func NewHostMem(virtualMemory memory.VirtualMemory) *hostMem {
	m := new(hostMem)
	m.vm = virtualMemory
	nrPages := virtualMemory.Size()
	m.accessMap = make([]byte, nrPages)
	m.faultListeners = make([]memory.FaultListener, 0)
	m.vm.AccessRightsDisabled(true)
	m.RWMutex = &sync.RWMutex{}
	return m
}

func (m *Multiview) Leave() {
	m.conn.Close()
}

func (m *Multiview) Shutdown() {
	m.conn.Close()
	m.
}

func (m *Multiview) Join(memSize, pageByteSize int) error {
	c := make(chan bool)
	//handler for all incoming messages in the host process, ie. read/write requests/replies, and invalidation requests.
	handler := func(message network.Message) error {
		var msg network.MultiviewMessage
		switch message.(type) {
		case network.SimpleMessage:
			msg = network.MultiviewMessage{From: message.GetFrom(), To: message.GetTo(), Type: message.GetType()}
		case network.MultiviewMessage:
			msg = message.(network.MultiviewMessage)
		}
		return m.messageHandler(msg, c)
	}
	client := network.NewP2PClient(handler)
	err := m.StartAndConnect(memSize, pageByteSize, client)
	panicOnErr(err)
	<-c
	log.Println("host joined network with id: ", m.Id)
	return err
}

func (m *Multiview) Initialize(memSize, pageByteSize int, nrProcs int) error {
	var err error
	filename := "BenchmarkResults/multivewLog" + strings.Replace(strings.Replace(time.Now().String()[:19], " ", "_", -1), ":", "-", -1) + ".csv"
	f, err := os.Create(filename)
	if err != nil {
		f.Close()
		log.Fatal(err)
	}
	m.csvLogger = network.NewCSVStructLogger(f)
	time.Sleep(time.Millisecond * 100)
	vm := memory.NewVmem(memSize, pageByteSize)
	bm := treadmarks.NewBarrierManagerImp(nrProcs)
	lm := treadmarks.NewLockManagerImp()
	manager := NewUpdatedManager(vm, lm, bm)
	manager.Connect("localhost:2000")
	return m.Join(memSize, pageByteSize)
}

func (m *Multiview) StartAndConnect(memSize, pageByteSize int, client network.IClient) error {
	vm := memory.NewVmem(memSize, pageByteSize)
	m.mem = NewHostMem(vm)
	for i := 0; i < max(memSize, pageByteSize)/pageByteSize; i++ {
		m.setInAccessMap(i, memory.READ_WRITE)
	}
	m.conn = client
	m.mem.addFaultListener(m.onFault)
	for {
		if err := m.conn.Connect("localhost:2000"); err != nil {
			time.Sleep(time.Millisecond * 100)
		} else {
			break
		}
	}
	return nil
}

func (t *Multiview) ReadInt(addr int) int {

	b, _ := t.ReadBytes(addr, 4)
	result, _ := binary.Varint(b)
	return int(result)
}

func (t *Multiview) ReadInt64(addr int) int {

	b, _ := t.ReadBytes(addr, 8)
	result, _ := binary.Varint(b)
	return int(result)
}

func (t *Multiview) WriteBytes(addr int, val []byte) error {
	var err error = nil
	for b, val := range val {
		err = t.Write(addr+b, val)
	}
	return err
}

func (t *Multiview) WriteInt(addr int, i int) {
	buff := make([]byte, 4)
	_ = binary.PutVarint(buff, int64(i))
	if len(buff) != 4 {
		panic("wrong length of buffer! Expected 4, got" + string(len(buff)))
	}
	t.WriteBytes(addr, buff)
}

func (t *Multiview) WriteInt64(addr int, i int) {
	buff := make([]byte, 8)
	_ = binary.PutVarint(buff, int64(i))
	if len(buff) != 8 {
		panic("wrong length of buffer! Expected 4, got" + string(len(buff)))
	}
	t.WriteBytes(addr, buff)
}

func (m *Multiview) Lock(id int) {
	//only send lock request if I don't already have it.
	if m.hasLock[id] {
		return
	}
	c := make(chan string)
	m.sequenceNumber++
	i := m.sequenceNumber
	m.chanMap[i] = c
	msg := network.MultiviewMessage{
		Type:    LOCK_ACQUIRE_REQUEST,
		From:    m.Id,
		To:      byte(0),
		Id:      id,
		EventId: i,
	}
	m.conn.Send(msg)
	<-c
	m.hasLock[id] = true
	m.chanMap[i] = nil
}

func (m *Multiview) Release(id int) {
	msg := network.MultiviewMessage{
		Type: LOCK_RELEASE,
		From: m.Id,
		To:   byte(0),
		Id:   id,
	}
	m.hasLock[id] = false
	m.conn.Send(msg)
}

func (m *Multiview) Barrier(id int) {
	c := make(chan string)
	m.sequenceNumber++
	i := m.sequenceNumber
	m.chanMap[i] = c
	msg := network.MultiviewMessage{
		Type:    BARRIER_REQUEST,
		From:    m.Id,
		To:      byte(0),
		Id:      id,
		EventId: i,
	}
	m.conn.Send(msg)
	<-c
	m.chanMap[i] = nil
}

func (m *hostMem) translateAddr(addr int) int {
	return addr % m.vm.Size()
}

func (m *hostMem) getVPageNr(addr int) int {
	return addr / m.vm.GetPageSize()
}

func (m *Multiview) Read(addr int) (byte, error) {
	if m.getInAccessMap(m.mem.getVPageNr(addr)) == memory.NO_ACCESS {
		for _, l := range m.mem.faultListeners {
			l(addr, 1, 0, "READ", []byte{0})
		}
	}
	res, _ := m.mem.vm.Read(m.mem.translateAddr(addr))
	return res, nil
}

func (m *Multiview) ReadBytes(addr, length int) ([]byte, error) {
	result := make([]byte, length)
	for i := range result {
		result[i], _ = m.Read(addr + i)
	}
	return result, nil
}

func (m *Multiview) privilegedRead(addr, length int) ([]byte, error) {
	result := make([]byte, length)
	result = m.mem.vm.PrivilegedRead(addr, length)
	return result, nil
}

/*
func (m *Multiview) ReadBytes(addr, length int) ([]byte, error) {
	result := make([]byte, length)
	//check access rights
	for i := addr; i < addr+length; i += m.mem.vm.GetPageSize() {
		if m.getInAccessMap(m.mem.getVPageNr(addr)) == memory.NO_ACCESS {
			return nil, errors.New("Access Denied")
		}
	}
	return m.mem.vm.ReadBytes(m.mem.translateAddr(addr), length)
}*/

func (m *Multiview) Write(addr int, val byte) error {
	if m.getInAccessMap(m.mem.getVPageNr(addr)) != memory.READ_WRITE {
		for _, l := range m.mem.faultListeners {
			l(addr, 1, 1, "WRITE", []byte{val})
		}
	}
	return m.mem.vm.Write(m.mem.translateAddr(addr), val)
}

func (m *Multiview) Malloc(sizeInBytes int) (int, error) {
	c := make(chan string)
	m.sequenceNumber++
	i := m.sequenceNumber
	m.chanMap[i] = c
	msg := network.MultiviewMessage{
		Type:          MALLOC_REQUEST,
		From:          m.Id,
		To:            byte(0),
		EventId:       i,
		Minipage_size: sizeInBytes, //<- contains the size for the allocation!
	}
	m.conn.Send(msg)
	s := <-c
	m.chanMap[i] = nil
	res, err := strconv.Atoi(s)
	if err != nil {
		return -1, errors.New(s)
	}
	return res, nil
}

func (m *Multiview) MultiMalloc(sizes []int) ([]int, error) {
	c := make(chan string)
	m.sequenceNumber++
	i := m.sequenceNumber
	m.chanMap[i] = c
	msg := network.MultiviewMessage{
		Type:    MULTI_MALLOC_REQUEST,
		From:    m.Id,
		To:      byte(0),
		EventId: i,
		IntArr:  sizes, //<- contains the sizes for the allocations
	}
	m.conn.Send(msg)
	s := <-c
	m.chanMap[i] = nil
	return StringOfIntsToIntArray(s), nil
}

func (m *Multiview) Free(pointer, length int) error {
	c := make(chan string)
	m.sequenceNumber++
	i := m.sequenceNumber
	m.chanMap[i] = c
	msg := network.MultiviewMessage{
		Type:          FREE_REQUEST,
		From:          m.Id,
		To:            byte(0),
		EventId:       i,
		Fault_addr:    pointer,
		Minipage_size: length, //<- length here
	}
	m.conn.Send(msg)
	res := <-c
	m.chanMap[i] = nil
	if res != "ok" {
		return errors.New(res)
	}
	return nil
}

func (m *Multiview) GetPageSize() int {
	return m.mem.vm.GetPageSize()
}

func (m *Multiview) GetMemoryByteSize() int {
	return m.mem.vm.Size()
}

func (m *hostMem) addFaultListener(l memory.FaultListener) {
	m.faultListeners = append(m.faultListeners, l)
}

//ID's are placeholder values waiting for integration. faultType = memory.READ_REQUEST OR memory.WRITE_REQUEST
func (m *Multiview) onFault(addr int, length int, faultType byte, accessType string, value []byte) error {
	str := ""
	if faultType == 0 {
		str = READ_REQUEST
	} else if faultType == 1 {
		str = WRITE_REQUEST
	}
	c := make(chan string)
	m.sequenceNumber++
	i := m.sequenceNumber
	m.chanMap[i] = c
	msg := network.MultiviewMessage{
		Type:       str,
		From:       m.Id,
		To:         byte(0),
		EventId:    m.sequenceNumber,
		Fault_addr: addr,
	}
	err := m.conn.Send(msg)
	panicOnErr(err)
	<-c
	m.chanMap[i] = nil
	//send ack
	msg = network.MultiviewMessage{
		From:       m.Id,
		To:         byte(0),
		Fault_addr: addr,
	}
	if faultType == 0 {
		msg.Type = READ_ACK
	} else if faultType == 1 {
		msg.Type = WRITE_ACK
	}
	m.conn.Send(msg)
	return nil
}

func (m *Multiview) messageHandler(msg network.MultiviewMessage, c chan bool) error {
	log.Println("received message at host", m.Id, msg)
	switch msg.Type {
	case WELCOME_MESSAGE:
		m.Id = msg.To
		c <- true
	case READ_REPLY, WRITE_REPLY:
		privBase := msg.Privbase
		//write data to privileged view, ie. the actual memory representation
		for i, byt := range msg.Data {
			err := m.Write(privBase+i, byt)
			if err != nil {
				log.Println("failed to write to privileged view at addr: ", privBase+i, " with error: ", err)
				break
			}
		}
		var right byte
		if msg.Type == READ_REPLY {
			right = memory.READ_ONLY
		} else {
			right = memory.READ_WRITE
		}
		m.setInAccessMap(m.mem.getVPageNr(msg.Fault_addr), right)
		m.chanMap[msg.EventId] <- "done" //let the blocking caller resume their work
	case READ_REQUEST, WRITE_REQUEST:
		vpagenr := m.mem.getVPageNr(msg.Fault_addr)
		if msg.Type == READ_REQUEST && m.getInAccessMap(vpagenr) == memory.READ_WRITE {
			m.setInAccessMap(vpagenr, memory.READ_ONLY)
		} else if msg.Type == WRITE_REQUEST {
			m.setInAccessMap(vpagenr, memory.NO_ACCESS)

		}
		if msg.Type == READ_REQUEST {
			msg.Type = READ_REPLY
		} else {
			msg.Type = WRITE_REPLY
		}
		//send reply back to requester including data
		msg.To = msg.From
		res, err := m.ReadBytes(msg.Privbase, msg.Minipage_size)
		panicOnErr(err)
		msg.Data = res
		m.conn.Send(msg)

	case INVALIDATE_REQUEST:
		m.setInAccessMap(m.mem.getVPageNr(msg.Fault_addr), memory.NO_ACCESS)
		msg.Type = INVALIDATE_REPLY
		msg.To = byte(0)
		m.conn.Send(msg)
	case MALLOC_REPLY:
		if msg.Err != "" {
			m.chanMap[msg.EventId] <- msg.Err
		} else {
			s := msg.Fault_addr
			m.chanMap[msg.EventId] <- strconv.Itoa(s)
		}
	case MULTI_MALLOC_REPLY:
		if msg.Err != "" {
			m.chanMap[msg.EventId] <- msg.Err
		} else {
			m.chanMap[msg.EventId] <- arrayToString(msg.IntArr, ",")
		}
	case FREE_REPLY:
		if msg.Err != "" {
			m.chanMap[msg.EventId] <- msg.Err
		} else {
			m.chanMap[msg.EventId] <- "ok"
		}
	case LOCK_ACQUIRE_RESPONSE:
		m.chanMap[msg.EventId] <- "ok"
	case BARRIER_RESPONSE:
		m.chanMap[msg.EventId] <- "ok"
	}
	return nil
}

func (m *Multiview) CSVLoggingIsEnabled(b bool) {
	if b == true {
		m.csvLogger.Enable()
	} else {
		m.csvLogger.Disable()
	}
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func arrayToString(a []int, delim string) string {
	return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
	//return strings.Trim(strings.Join(strings.Split(fmt.Sprint(a), " "), delim), "[]")
	//return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(a)), delim), "[]")
}

func StringOfIntsToIntArray(s string) []int {
	stringSlice := strings.Split(s, ",")
	var err error
	var res []int = make([]int, len(stringSlice))
	for i, s := range stringSlice {
		res[i], err = strconv.Atoi(s)
		panicOnErr(err)
	}
	return res
}

func (m *Multiview) SetShouldLogNetwork(b bool) {
	m.shouldLogNetwork = b
	if m.messagesSent == nil {
		m.messagesSent = make([]int, 12)
	}
}

func (m *Multiview) LogMessage(message network.MultiviewMessage) {
	if m.shouldLogNetwork {

	}
}

func mTypeToInt(t string) int {
	switch t {
	case READ_REQUEST:
		return 0
	case WRITE_REQUEST:
		return 1
	case INVALIDATE_REPLY:
		return 2
	case MALLOC_REQUEST:
		return 3
	case FREE_REQUEST:
		return 4
	case WRITE_ACK:
		return 5
	case READ_ACK:
		return 6
	case LOCK_ACQUIRE_REQUEST:
		return 7
	case BARRIER_REQUEST:
		return 8
	case LOCK_RELEASE:
		return 9
	case MULTI_MALLOC_REQUEST:
		return 10
	case MULTI_MALLOC_REPLY :
		return 11
	}
}
