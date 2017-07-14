package multiview

import (
	"testing"
	"DSM-project/network"
	"DSM-project/memory"
	"github.com/stretchr/testify/assert"
	"time"
)


func TestHandlerREADWRITE_REPLY(t *testing.T) {
	mw := NewMultiView()

	mw.chanMap = make(map[byte]chan string)
	cMock := NewClientMock()
	mw.StartAndConnect(4096, 128, cMock)
	msg := network.MultiviewMessage{
		Type: WELCOME_MESSAGE,
		To: byte(9),
	}
	c := make(chan bool)
	go mw.messageHandler(msg, c)
	<- c
	assert.Equal(t, byte(9), mw.id)
	//test read_reply, ie. from a read request
	channel := make(chan string)
	mw.chanMap[100] = channel
	msg = network.MultiviewMessage{
		Type: READ_REPLY,
		Data: []byte{byte(1), byte(2), byte(3)},
		Privbase: 100,
		Fault_addr: 4096+100,
		EventId: 100,
	}
	go mw.messageHandler(msg, nil)
	<- channel
	assert.Equal(t, memory.READ_ONLY, mw.mem.accessMap[mw.mem.getVPageNr(4096+100)])
	res, err := mw.Read(100 + 4096)
	assert.Nil(t, err)
	assert.Equal(t, byte(1), res)

	res, err = mw.Read(101 + 4096)
	assert.Nil(t, err)
	assert.Equal(t, byte(2), res)

	res, err = mw.Read(102 + 4096)
	assert.Nil(t, err)
	assert.Equal(t, byte(3), res)
}

func TestHandlerREADWRITE_REQ(t *testing.T) {
	mw := NewMultiView()

	cMock := NewClientMock()
	mw.StartAndConnect(4096, 128, cMock)
	mw.Write(105, byte(12))
	msg := network.MultiviewMessage{
		Type: WRITE_REQUEST,
		To: byte(9),
		From: byte(8),
		Privbase: 100,
		Minipage_base: 5,
		Minipage_size: 25,
		Fault_addr: 100 + 4096 + 5,
	}
	mw.messageHandler(msg,nil)
	assert.Equal(t, 1, len(cMock.messages))
	assert.Equal(t, byte(12), cMock.messages[0].Data[5])
	assert.Equal(t, msg.From, cMock.messages[0].To)
	assert.Equal(t, memory.NO_ACCESS, mw.mem.accessMap[mw.mem.getVPageNr(100 + 4096 + 5)])
}

func TestHandlerINVALIDATE(t *testing.T) {
	mw := NewMultiView()

	cMock := NewClientMock()
	mw.StartAndConnect(4096, 128, cMock)
	msg := network.MultiviewMessage{
		Type: INVALIDATE_REQUEST,
		Fault_addr: 255 + 4096,
	}
	mw.messageHandler(msg,nil)
	reply := cMock.messages[0]
	assert.Equal(t, INVALIDATE_REPLY, reply.Type)
	assert.Equal(t, memory.NO_ACCESS, mw.mem.accessMap[mw.mem.getVPageNr(255 + 4096)])
	go func() {
		mw.Read(255 + 4096)
	}()
	time.Sleep(time.Millisecond * 200)
	mw.chanMap[cMock.messages[1].EventId] <- "ok"
	time.Sleep(time.Millisecond * 200)
}

func TestHostMem_WriteAndRead(t *testing.T) {
	mw := NewMultiView()
	cMock := NewClientMock()
	mw.StartAndConnect(4096, 128, cMock)
	//test that an access miss fires a read/write request to the manager
	go func() {
		mw.Read(4096 + 100)
	}()
	time.Sleep(time.Millisecond * 200)
	reply := cMock.messages[0]
	mw.chanMap[reply.EventId] <- "ok"
	assert.Equal(t, 4096 + 100, reply.Fault_addr )
	assert.Equal(t, READ_REQUEST, reply.Type)
	time.Sleep(time.Millisecond * 200)
	assert.Equal(t, READ_ACK, cMock.messages[1].Type)

	go func() {
		mw.Write(4096 + 100, byte(99))
	}()
	time.Sleep(time.Millisecond * 200)
	reply = cMock.messages[2]
	mw.chanMap[reply.EventId] <- "ok"
	assert.Equal(t, WRITE_REQUEST, reply.Type)
	time.Sleep(time.Millisecond * 200)
	assert.Equal(t, WRITE_ACK, cMock.messages[3].Type)

}

type clientMock struct {
	messages []network.MultiviewMessage
	handler func(msg network.MultiviewMessage)
}

func (c *clientMock) Send(message network.Message) error {
	msg := message.(network.MultiviewMessage)
	c.messages = append(c.messages, msg)
	return nil
}

func (c *clientMock) GetTransciever() network.ITransciever {
	panic("implement me")
}

func (c *clientMock) Connect(address string) error {
	return nil
}

func (c *clientMock) Close() {
}


func NewClientMock() *clientMock {
	cMock := new(clientMock)
	cMock.messages = make([]network.MultiviewMessage, 0)

	return cMock
}