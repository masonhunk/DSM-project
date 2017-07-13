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

type Multiview struct {
	conn network.IClient
	mem *hostMem
	id byte
	server network.Server
	chanMap map[byte]chan string
	sequenceNumber byte
}

type hostMem struct {
	vm	memory.VirtualMemory
	accessMap map[int]byte //key = vpage number, value, access right
	faultListeners []memory.FaultListener
}

func NewMultiView() *Multiview {
	return new(Multiview)
}

func NewHostMem(virtualMemory memory.VirtualMemory) *hostMem {
	m := new(hostMem)
	m.vm = virtualMemory
	m.accessMap = make(map[int]byte)
	m.faultListeners = make([]memory.FaultListener, 0)
	m.vm.AccessRightsDisabled(true)
	return m
}

func (m *Multiview) Leave() {
	m.conn.Close()
}

func (m *Multiview) Shutdown() {
	m.conn.Close()
	m.server.StopServer()
}

func (m *Multiview) Join(memSize, pageByteSize int) error {
	c := make(chan bool)
	m.chanMap = make(map[byte]chan string)
	//handler for all incoming messages in the host process, ie. read/write requests/replies, and invalidation requests.
	msgHandler := func(msg network.Message) error {
		switch msg.Type {
		case WELCOME_MESSAGE:
			m.id = msg.To
			c <- true
		case READ_REPLY, WRITE_REPLY:
			privBase := msg.Privbase
			//write data to privileged view, ie. the actual memory representation
			for i, byt := range msg.Data {
				err := m.Write(privBase + i, byt)
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
			m.mem.accessMap[m.mem.getVPageNr(msg.Fault_addr)] = right
			m.chanMap[msg.EventId] <- "done" //let the blocking caller resume their work
		case READ_REQUEST, WRITE_REQUEST:
			vpagenr := m.mem.getVPageNr(msg.Fault_addr)
			if msg.Type == READ_REQUEST && m.mem.accessMap[vpagenr] == memory.READ_WRITE &&
					vpagenr >= m.mem.vm.Size()/m.mem.vm.GetPageSize() {
				m.mem.accessMap[vpagenr] = memory.READ_ONLY
				msg.Type = READ_REPLY
			} else if msg.Type == WRITE_REQUEST && vpagenr >= m.mem.vm.Size()/m.mem.vm.GetPageSize() {
				m.mem.accessMap[vpagenr] = memory.NO_ACCESS
				msg.Type = WRITE_REPLY
			}
			//send reply back to requester including data
			msg.To = msg.From
			res, err := m.ReadBytes(msg.Privbase, msg.Minipage_size)
			if err != nil {
				fmt.Println(err)
			} else {
				msg.Data = res
				m.conn.Send(msg)
			}
		case INVALIDATE_REQUEST:
			m.mem.accessMap[m.mem.getVPageNr(msg.Fault_addr)] = memory.NO_ACCESS
			msg.Type = INVALIDATE_REPLY
			m.conn.Send(msg)
		case MALLOC_REPLY:
			if msg.Err != nil {
				m.chanMap[msg.EventId] <- msg.Err.Error()
			} else {
				s := msg.Fault_addr
				m.chanMap[msg.EventId] <- strconv.Itoa(s)
			}
		case FREE_REPLY:
			if msg.Err != nil {
				m.chanMap[msg.EventId] <- msg.Err.Error()
			} else {
				m.chanMap[msg.EventId] <- "ok"
			}
		}
		return nil
	}

	client := network.NewClient(msgHandler)
	err := m.StartAndConnect(memSize, pageByteSize, client)
	<- c
	return err
}

func (m *Multiview) Initialize(memSize, pageByteSize int) error {
	var err error
	m.server, err = network.NewServer(func(message network.Message) error {return nil}, "2000")
	if err != nil {
		return err
	}
	time.Sleep(time.Millisecond * 100)
	vm := memory.NewVmem(memSize, pageByteSize)
	manager := NewManager(vm)
	manager.Connect("localhost:2000")
	return m.Join(memSize, pageByteSize)
}

func (m *Multiview) StartAndConnect(memSize, pageByteSize int, client network.IClient) error {
	vm := memory.NewVmem(memSize, pageByteSize)
	m.mem = NewHostMem(vm)
	for i := 0; i < memSize/pageByteSize; i++ {
		m.mem.accessMap[i] = memory.READ_WRITE
	}
	m.conn = client
	m.mem.addFaultListener(m.onFault)
	return m.conn.Connect("localhost:2000")
}


func (m *hostMem) translateAddr(addr int) int {
	return addr % m.vm.Size()
}

func (m *hostMem) getVPageNr(addr int) int {
	return addr/m.vm.GetPageSize()
}

func (m *Multiview) Read(addr int) (byte, error) {
	if m.mem.accessMap[m.mem.getVPageNr(addr)] == 0 {
		for _, l := range m.mem.faultListeners {
			l(addr, 0)
		}
		return 0, memory.AccessDeniedErr
	}
	res, _ := m.mem.vm.Read(m.mem.translateAddr(addr))
	return res, nil
}

func (m *Multiview) ReadBytes(addr, length int) ([]byte, error) {
	//check access rights
	for i := addr; i < addr + length; i += m.mem.vm.GetPageSize() {
		if m.mem.accessMap[m.mem.getVPageNr(addr)] == memory.NO_ACCESS {
			return nil, errors.New("Access Denied")
		}
	}
	return m.mem.vm.ReadBytes(m.mem.translateAddr(addr), length)
}

func (m *Multiview) Write(addr int, val byte) error {
	if m.mem.accessMap[m.mem.getVPageNr(addr)] != memory.READ_WRITE {
		for _, l := range m.mem.faultListeners {
			l(addr, 1)
		}
		return memory.AccessDeniedErr
	}
	return m.mem.vm.Write(m.mem.translateAddr(addr), val)
}

func (m *Multiview) Malloc(sizeInBytes int) (int, error) {
	c := make(chan string)
	m.chanMap[m.sequenceNumber] = c
	msg := network.Message{
		Type: MALLOC_REQUEST,
		From: m.id,
		To: byte(1),
		EventId: m.sequenceNumber,
		Minipage_size: sizeInBytes, //<- contains the size for the allocation!
	}
	m.conn.Send(msg)
	s := <- c
	m.chanMap[m.sequenceNumber] = nil
	m.sequenceNumber++
	res, err := strconv.Atoi(s)
	if err != nil {
		return 0, errors.New(s)
	}
	return res, nil
}

func (m *Multiview) Free(pointer, length int) error {
	c := make(chan string)
	m.chanMap[m.sequenceNumber] = c
	msg := network.Message{
		Type: FREE_REQUEST,
		From: m.id,
		To: byte(1),
		EventId: m.sequenceNumber,
		Fault_addr: pointer,
		Minipage_size: length, //<- length here
	}
	m.conn.Send(msg)
	res := <- c
	m.chanMap[m.sequenceNumber] = nil
	m.sequenceNumber++
	if res != "ok" {
		return errors.New(res)
	}
	return nil
}

func (m *hostMem) addFaultListener(l memory.FaultListener) {
	m.faultListeners = append(m.faultListeners, l)
}

//ID's are placeholder values waiting for integration. faultType = memory.READ_REQUEST OR memory.WRITE_REQUEST
func (m *Multiview) onFault(addr int, faultType byte) {
	str := ""
	if faultType == 0 {
		str = READ_REQUEST
	} else if faultType == 1 {
		str = WRITE_REQUEST
	}
	c := make(chan string)
	m.chanMap[m.sequenceNumber] = c
	msg := network.Message{
		Type: str,
		From: m.id,
		To: byte(1),
		EventId: m.sequenceNumber,
		Fault_addr: addr,
	}
		err := m.conn.Send(msg)
	if err == nil {
		<- c
		m.chanMap[m.sequenceNumber] = nil
		m.sequenceNumber++
		//send ack
		msg := network.Message{
			From: m.id,
			To: byte(1),
		}
		if faultType == 0 {
			msg.Type = READ_ACK
		} else if faultType == 1 {
			msg.Type = WRITE_ACK
		}
		m.conn.Send(msg)
	} else {
		fmt.Println(err)
	}

}
