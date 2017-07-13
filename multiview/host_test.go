package multiview

import (
	"testing"
	"DSM-project/network"
	"fmt"
	"DSM-project/memory"
	"strconv"
	"github.com/stretchr/testify/assert"
	"time"
)


func TestHandlerREADWRITE_REPLY(t *testing.T) {
	chanMap = make(map[byte]chan string)
	cMock := NewClientMock()
	StartAndConnect(4096, 128, cMock)
	msg := network.Message{
		Type: WELCOME_MESSAGE,
		To: byte(9),
	}
	cMock.handler(msg)
	assert.Equal(t, byte(9), id)
	//test read_reply, ie. from a read request
	channel := make(chan string)
	chanMap[100] = channel
	msg = network.Message{
		Type: READ_REPLY,
		Data: []byte{byte(1), byte(2), byte(3)},
		Privbase: 100,
		Fault_addr: 4096+100,
		EventId: 100,
	}
	go cMock.handler(msg)
	<- channel
	assert.Equal(t, memory.READ_ONLY, mem.accessMap[mem.getVPageNr(4096+100)])
	res, err := mem.Read(100 + 4096)
	assert.Nil(t, err)
	assert.Equal(t, byte(1), res)

	res, err = mem.Read(101 + 4096)
	assert.Nil(t, err)
	assert.Equal(t, byte(2), res)

	res, err = mem.Read(102 + 4096)
	assert.Nil(t, err)
	assert.Equal(t, byte(3), res)
}

func TestHandlerREADWRITE_REQ(t *testing.T) {
	cMock := NewClientMock()
	StartAndConnect(4096, 128, cMock)
	mem.Write(105, byte(12))
	msg := network.Message{
		Type: WRITE_REQUEST,
		To: byte(9),
		From: byte(8),
		Privbase: 100,
		Minipage_base: 5,
		Minipage_size: 25,
		Fault_addr: 100 + 4096 + 5,
	}
	cMock.handler(msg)
	assert.Equal(t, 1, len(cMock.messages))
	assert.Equal(t, byte(12), cMock.messages[0].Data[5])
	assert.Equal(t, msg.From, cMock.messages[0].To)
	assert.Equal(t, memory.NO_ACCESS, mem.accessMap[mem.getVPageNr(100 + 4096 + 5)])
}

func TestHandlerINVALIDATE(t *testing.T) {
	cMock := NewClientMock()
	StartAndConnect(4096, 128, cMock)
	msg := network.Message{
		Type: INVALIDATE_REQUEST,
		Fault_addr: 255 + 4096,
	}
	cMock.handler(msg)
	reply := cMock.messages[0]
	assert.Equal(t, INVALIDATE_REPLY, reply.Type)
	assert.Equal(t, memory.NO_ACCESS, mem.accessMap[mem.getVPageNr(255 + 4096)])
	go func() {
		_, err := mem.Read(255 + 4096)
		assert.NotNil(t, err)
	}()
	time.Sleep(time.Millisecond * 200)
	chanMap[cMock.messages[1].EventId] <- "ok"
}

func TestHostMem_WriteAndRead(t *testing.T) {
	cMock := NewClientMock()
	StartAndConnect(4096, 128, cMock)

	//test that an access miss fires a read/write request to the manager
	go func() {
		_, err := mem.Read(4096 + 100)
		assert.NotNil(t, err)
	}()
	time.Sleep(time.Millisecond * 200)
	reply := cMock.messages[0]
	chanMap[reply.EventId] <- "ok"
	assert.Equal(t, 4096 + 100, reply.Fault_addr )
	assert.Equal(t, READ_REQUEST, reply.Type)
	time.Sleep(time.Millisecond * 200)
	assert.Equal(t, READ_ACK, cMock.messages[1].Type)

	go func() {
		err := mem.Write(4096 + 100, byte(99))
		assert.NotNil(t, err)
	}()
	time.Sleep(time.Millisecond * 200)
	reply = cMock.messages[2]
	chanMap[reply.EventId] <- "ok"
	assert.Equal(t, WRITE_REQUEST, reply.Type)
	time.Sleep(time.Millisecond * 200)
	assert.Equal(t, WRITE_ACK, cMock.messages[3].Type)

}

type clientMock struct {
	messages []network.Message
	handler func(msg network.Message)
}

func (c *clientMock) GetTransciever() network.ITransciever {
	panic("implement me")
}

func (c *clientMock) Connect(address string) error {
	return nil
}

func (c *clientMock) Close() {
}

func (c *clientMock) Send(message network.Message) error {
	c.messages = append(c.messages, message)
	return nil
}

func NewClientMock() *clientMock {
	cMock := new(clientMock)
	cMock.messages = make([]network.Message, 0)
	msgHandler := func(msg network.Message) {
		switch msg.Type {
		case WELCOME_MESSAGE:
			id = msg.To
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
				fmt.Println("test")
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
				s := msg.Minipage_base
				chanMap[msg.EventId] <- strconv.Itoa(s)
			}
		case FREE_REPLY:
			if msg.Err != nil {
				chanMap[msg.EventId] <- msg.Err.Error()
			} else {
				chanMap[msg.EventId] <- "ok"
			}
		}
	}
	cMock.handler = msgHandler
	return cMock
}