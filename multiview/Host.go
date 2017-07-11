package multiview

import (
	"DSM-project/network"
	"DSM-project/memory"
	"fmt"
	"strconv"
	"errors"
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
)

var conn network.Client
var mem *hostMem

type hostMem struct {
	vm	memory.VirtualMemory
	accessMap map[int]byte
	faultListeners []memory.FaultListener
}

func NewHostMem(virtualMemory memory.VirtualMemory) *hostMem {
	m := new(hostMem)
	m.vm = virtualMemory
	m.accessMap = make(map[int]byte) //key = vpage number, value, access right
	m.faultListeners = make([]memory.FaultListener, 0)
	m.vm.AccessRightsDisabled(true)
	return m
}

func (m *hostMem) translateAddr(addr int) int {
	return addr % m.vm.Size()
}

func (m *hostMem) getVPageNr(addr int) int {
	return addr % m.vm.GetPageSize()
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
	panic("implement me")
}

func (m *hostMem) Write(addr int, val byte) error {
	if m.accessMap[m.getVPageNr(addr)] != memory.READ_WRITE {
		for _, l := range m.faultListeners {
			l(addr, 0)
		}
		return memory.AccessDeniedErr
	}
	return m.vm.Write(m.translateAddr(addr), val)
}

func (m *hostMem) Malloc(sizeInBytes int) (int, error) {
	c := make(chan string)

	msg := network.Message{
		Type: MALLOC_REQUEST,
		From: byte(1),
		To: byte(1),
		Data: nil,
		Err: nil,
		Event: &c,
		Fault_addr: 0,
		Minipage_size: sizeInBytes, //<- contains the size for the allocation!
	}
	conn.Send(msg)
	s := <- c
	res, err := strconv.Atoi(s)
	if err != nil {
		return 0, errors.New(s)
	}
	return res, nil
}

func (m *hostMem) Free(pointer, length int) error {
	c := make(chan string)

	msg := network.Message{
		Type: FREE_REQUEST,
		From: byte(1),
		To: byte(1),
		Data: nil,
		Err: nil,
		Event: &c,
		Fault_addr: pointer,
		Minipage_size: length, //<- length here
	}
	conn.Send(msg)
	res := <- c
	if res != "ok" {
		return errors.New(res)
	}
	return nil
}

func (m *hostMem) addFaultListener(l memory.FaultListener) {
	m.faultListeners = append(m.faultListeners, l)
}


func Initialize(memSize, pageByteSize int) {
	vm := memory.NewVmem(memSize, pageByteSize)
	mem = NewHostMem(vm)
	//handler for all incoming messages in the host process, ie. read/write requests/replies, and invalidation requests.
	msgHandler := func(msg network.Message) {
		switch msg.Type {
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
			*msg.Event <- "done" //let the blocking caller resume their work
		case READ_REQUEST, WRITE_REQUEST:
			if msg.Type == READ_REQUEST && mem.accessMap[msg.Fault_addr] == memory.READ_WRITE {
				mem.accessMap[mem.getVPageNr(msg.Fault_addr)] = memory.READ_ONLY

			} else if msg.Type == WRITE_REQUEST {
				mem.accessMap[mem.getVPageNr(msg.Fault_addr)] = memory.NO_ACCESS
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
				*msg.Event <- msg.Err.Error()
			} else {
				s := msg.Fault_addr
				*msg.Event <- strconv.Itoa(s)
			}
		case FREE_REPLY:
			if msg.Err != nil {
				*msg.Event <- msg.Err.Error()
			} else {
				*msg.Event <- "ok"
			}
		}
	}

	conn = network.NewClient(msgHandler)
	conn.Connect("localhost:2000")
}

//ID's are placeholder values waiting for integration. faultType = memory.READ_REQUEST OR memory.WRITE_REQUEST
func onFault(addr int, faultType string) {
	c := make(chan string)
	msg := network.Message{
		Type: faultType,
		From: byte(1),
		To: byte(1),
		Data: nil,
		Err: nil,
		Event: &c,
		Fault_addr: addr,
	}
	err := conn.Send(msg)
	if err != nil {
		<- c
		//send ack
		//conn.Send()
	}

}
