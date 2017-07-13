package multiview

import (
	"DSM-project/network"
	"DSM-project/memory"
	"fmt"
	"strconv"
	"errors"
	"time"
)


const (
	READ_REQUEST = "RR"
	WRITE_REQUEST = "WR"
	READ_REPLY = "RRPL"
	WRITE_REPLY = "WRPL"
	INVALIDATE_REPLY = "INV"
	INVALIDATE_REQUEST = "INVQ"
	MALLOC_REQUEST = "MR"
	FREE_REQUEST = "FR"
	MALLOC_REPLY = "MRPL"
	FREE_REPLY = "FRPL"
	WELCOME_MESSAGE = "WELC"
	READ_ACK = "RA"
	WRITE_ACK = "WA"
)

var conn network.IClient
var mem *hostMem
var id byte
var server network.Server
var chanMap map[byte]chan string
var sequenceNumber byte = 0

type hostMem struct {
	vm	memory.VirtualMemory
	accessMap map[int]byte //key = vpage number, value, access right
	faultListeners []memory.FaultListener
}

func NewHostMem(virtualMemory memory.VirtualMemory) *hostMem {
	m := new(hostMem)
	m.vm = virtualMemory
	m.accessMap = make(map[int]byte)
	m.faultListeners = make([]memory.FaultListener, 0)
	m.vm.AccessRightsDisabled(true)
	return m
}

func Leave() {
	conn.Close()
}

func Shutdown() {
	conn.Close()
	server.StopServer()
}

func Join(memSize, pageByteSize int) error {
	c := make(chan bool)
	chanMap = make(map[byte]chan string)
	//handler for all incoming messages in the host process, ie. read/write requests/replies, and invalidation requests.
	msgHandler := func(msg network.Message) error {
		switch msg.Type {
		case WELCOME_MESSAGE:
			id = msg.To
			c <- true
		case READ_REPLY, WRITE_REPLY:
			privBase := msg.Privbase
			//write data to privileged view, ie. the actual memory representation
			for i, byt := range msg.Data {
				err := mem.Write(privBase + i, byt)
				if err != nil {
					fmt.Println("failed to write to privileged view at addr: ", privBase + i, " with error: ", err)
					break
				}
			}
			var right byte
			if msg.Type == READ_REPLY {
				right = memory.READ_ONLY
			} else {
				right = memory.READ_WRITE
			}
			mem.accessMap[mem.getVPageNr(msg.Fault_addr)] = right
			chanMap[msg.EventId] <- "done" //let the blocking caller resume their work
		case READ_REQUEST, WRITE_REQUEST:
			vpagenr := mem.getVPageNr(msg.Fault_addr)
			if msg.Type == READ_REQUEST && mem.accessMap[vpagenr] == memory.READ_WRITE && vpagenr >= mem.vm.Size()/mem.vm.GetPageSize() {
				mem.accessMap[vpagenr] = memory.READ_ONLY
				msg.Type = READ_REPLY
			} else if msg.Type == WRITE_REQUEST && vpagenr >= mem.vm.Size()/mem.vm.GetPageSize() {
				mem.accessMap[vpagenr] = memory.NO_ACCESS
				msg.Type = WRITE_REPLY
			}
			//send reply back to requester including data
			msg.To = msg.From
			res, err := mem.ReadBytes(msg.Privbase, msg.Minipage_size)
			if err != nil {
				fmt.Println(err)
			} else {
				msg.Data = res
				conn.Send(msg)
			}
		case INVALIDATE_REQUEST:
			mem.accessMap[mem.getVPageNr(msg.Fault_addr)] = memory.NO_ACCESS
			msg.Type = INVALIDATE_REPLY
			conn.Send(msg)
		case MALLOC_REPLY:
			if msg.Err != nil {
				chanMap[msg.EventId] <- msg.Err.Error()
			} else {
				s := msg.Fault_addr
				chanMap[msg.EventId] <- strconv.Itoa(s)
			}
		case FREE_REPLY:
			if msg.Err != nil {
				chanMap[msg.EventId] <- msg.Err.Error()
			} else {
				chanMap[msg.EventId] <- "ok"
			}
		}
		return nil
	}

	client := network.NewClient(msgHandler)
	err := StartAndConnect(memSize, pageByteSize, client)
	<- c
	return err
}

func Initialize(memSize, pageByteSize int) error {
	var err error
	server, err = network.NewServer(func(message network.Message) error {return nil}, "2000")
	if err != nil {
		return err
	}
	time.Sleep(time.Millisecond * 100)
	vm := memory.NewVmem(memSize, pageByteSize)
	manager := NewManager(vm)
	manager.Connect("localhost:2000")
	return Join(memSize, pageByteSize)
}

func StartAndConnect(memSize, pageByteSize int, client network.IClient) error {
	vm := memory.NewVmem(memSize, pageByteSize)
	mem = NewHostMem(vm)
	for i := 0; i < memSize/pageByteSize; i++ {
		mem.accessMap[i] = memory.READ_WRITE
	}
	conn = client
	mem.addFaultListener(onFault)
	return conn.Connect("localhost:2000")
}


func (m *hostMem) translateAddr(addr int) int {
	return addr % m.vm.Size()
}

func (m *hostMem) getVPageNr(addr int) int {
	return addr/m.vm.GetPageSize()
}

func (m *hostMem) Read(addr int) (byte, error) {
	if m.accessMap[m.getVPageNr(addr)] == 0 {
		for _, l := range m.faultListeners {
			l(addr, 0)
		}
		return 0, memory.AccessDeniedErr
	}
	res, _ := m.vm.Read(m.translateAddr(addr))
	return res, nil
}

func (m *hostMem) ReadBytes(addr, length int) ([]byte, error) {
	//check access rights
	for i := addr; i < addr + length; i += mem.vm.GetPageSize() {
		if mem.accessMap[mem.getVPageNr(addr)] == memory.NO_ACCESS {
			return nil, errors.New("Access Denied")
		}
	}
	return m.vm.ReadBytes(mem.translateAddr(addr), length)
}

func (m *hostMem) Write(addr int, val byte) error {
	if m.accessMap[m.getVPageNr(addr)] != memory.READ_WRITE {
		for _, l := range m.faultListeners {
			l(addr, 1)
		}
		return memory.AccessDeniedErr
	}
	return m.vm.Write(m.translateAddr(addr), val)
}

func (m *hostMem) Malloc(sizeInBytes int) (int, error) {
	c := make(chan string)
	chanMap[sequenceNumber] = c
	msg := network.Message{
		Type: MALLOC_REQUEST,
		From: id,
		To: byte(1),
		EventId: sequenceNumber,
		Minipage_size: sizeInBytes, //<- contains the size for the allocation!
	}
	conn.Send(msg)
	s := <- c
	chanMap[sequenceNumber] = nil
	sequenceNumber++
	res, err := strconv.Atoi(s)
	if err != nil {
		return 0, errors.New(s)
	}
	return res, nil
}

func (m *hostMem) Free(pointer, length int) error {
	c := make(chan string)
	chanMap[sequenceNumber] = c
	msg := network.Message{
		Type: FREE_REQUEST,
		From: id,
		To: byte(1),
		EventId: sequenceNumber,
		Fault_addr: pointer,
		Minipage_size: length, //<- length here
	}
	conn.Send(msg)
	res := <- c
	chanMap[sequenceNumber] = nil
	sequenceNumber++
	if res != "ok" {
		return errors.New(res)
	}
	return nil
}

func (m *hostMem) addFaultListener(l memory.FaultListener) {
	m.faultListeners = append(m.faultListeners, l)
}

//ID's are placeholder values waiting for integration. faultType = memory.READ_REQUEST OR memory.WRITE_REQUEST
func onFault(addr int, faultType byte) {
	str := ""
	if faultType == 0 {
		str = READ_REQUEST
	} else if faultType == 1 {
		str = WRITE_REQUEST
	}
	c := make(chan string)
	chanMap[sequenceNumber] = c
	msg := network.Message{
		Type: str,
		From: id,
		To: byte(1),
		EventId: sequenceNumber,
		Fault_addr: addr,
	}
		err := conn.Send(msg)
	if err == nil {
		<- c
		chanMap[sequenceNumber] = nil
		sequenceNumber++
		//send ack
		msg := network.Message{
			From: id,
			To: byte(1),
		}
		if faultType == 0 {
			msg.Type = READ_ACK
		} else if faultType == 1 {
			msg.Type = WRITE_ACK
		}
		conn.Send(msg)
	} else {
		fmt.Println(err)
	}

}
