package tests

import (
	"testing"
	"DSM-project/network"
	"net"
	"fmt"
	"time"
	"github.com/stretchr/testify/assert"
	"sync"
)

var messages []network.Message
var mutex sync.Mutex

func messageHandler(message network.Message){
	mutex.Lock()
	messages = append(messages, message)
	mutex.Unlock()
	time.Sleep(time.Millisecond)
}

func createTransciever(conn net.Conn) {
	network.NewTransciever(conn, messageHandler)
}

func createEndpoint(port string) network.Endpoint{
	e, _ := network.NewEndpoint(port, createTransciever)
	return e
}

func TestEndpointCreationAndClose(t *testing.T) {
	e,_ := network.NewEndpoint("2000", func(conn net.Conn){return})
	e.Close()
}

func TestEndpointAndTranscieverConnectAndClose(t *testing.T){
	e,_ := network.NewEndpoint("2000", func(conn net.Conn){return})
	conn, _ := net.Dial("tcp", "localhost:2000")
	network.NewTransciever(conn, func(message network.Message){fmt.Println(message.Type)})
	e.Close()
}

func TestEndpointAndTranscieverConnectAndTalk(t *testing.T) {
	messages = []network.Message{}
	e := createEndpoint("2000")
	conn, _ := net.Dial("tcp", "localhost:2000")
	tr := network.NewTransciever(conn, messageHandler)
	tr.Send(network.Message{To: 10, Type: "Test", Data: []byte{1,2}})
	e.Close()
	time.Sleep(1000000000)
	assert.Len(t, messages, 1)
	assert.Equal(t, messages[0].Type, "Test")
	assert.Equal(t, messages[0].To, byte(10))
	assert.Equal(t, messages[0].Data, []byte{1,2})
}

func TestServerCreationWithMultipleClients(t *testing.T){
	messages = []network.Message{}
	s,_ := network.NewServer(func(message network.Message){}, "2000")
	c1 := network.NewClient(messageHandler)
	c1.Connect("localhost:2000")
	c2 := network.NewClient(messageHandler)
	c2.Connect("localhost:2000")
	c3 := network.NewClient(messageHandler)
	c3.Connect("localhost:2000")
	c1.Send(network.Message{To: byte(1), Type:  "Test", Data: []byte{1,3}})
	c2.Send(network.Message{To: byte(1), Type: "Test", Data: []byte{1,4}})
	c3.Send(network.Message{To: byte(1), Type: "Test", Data: []byte{1,5}})

	time.Sleep(1000000000)
	c1.Close()
	c2.Close()
	c3.Close()
	s.StopServer()
	fmt.Println(messages)

	assert.Len(t, messages, 3)
	assert.Len(t, s.Clients, 3)
}

func TestMultiMessages(t *testing.T) {
	messages = []network.Message{}
	s,_ := network.NewServer(messageHandler, "2000")
	c := network.NewClient(messageHandler)
	c.Connect("localhost:2000")
	time.Sleep(1000000000)
	c.Send(network.Message{To: 1, Type: "Test", Data: []byte{1,3}})
	c.Send(network.Message{To: 1, Type: "Test", Data: []byte{1,4}})
	c.Send(network.Message{To: 1, Type: "Test", Data: []byte{1,5}})
	time.Sleep(1000000000)
	c.Close()
	s.StopServer()
	assert.Len(t, messages, 6)
	assert.Len(t, s.Clients, 1)
}

func TestServerSending(t *testing.T) {
	messages = []network.Message{}
	s,_ := network.NewServer(func(message network.Message){}, "2000")
	c := network.NewClient(messageHandler)
	c.Connect("localhost:2000")
	for len(s.Clients) < 1{	}
	s.Send(network.Message{To: 1, Type: "Test", Data: []byte{1,2}})
	s.Send(network.Message{To: 1, Type: "Test", Data: []byte{1,2}})
	s.Send(network.Message{To: 1, Type: "Test", Data: []byte{1,2}})
	s.Send(network.Message{To: 1, Type: "Test", Data: []byte{1,2}})
	time.Sleep(1000000000)
	c.Close()
	s.StopServer()
	assert.Len(t, messages, 4)
	assert.Len(t, s.Clients, 1)
}

func TestClientToClient(t *testing.T) {
	m1 := []network.Message{}
	m2 := []network.Message{}
	s,_ := network.NewServer(messageHandler, "2000")
	c1 := network.NewClient(func(message network.Message){m1 = append(m1, message)})
	c1.Connect("localhost:2000")
	c2 := network.NewClient(func(message network.Message){m2 = append(m2, message)})
	c2.Connect("localhost:2000")

	c1.Send(network.Message{To: 2, Type: "From c1 to c2", Data: []byte{1,2}})
	c2.Send(network.Message{To: 1, Type: "From c2 to c1", Data: []byte{1,2}})

	time.Sleep(1000000000)
	c1.Close()
	c2.Close()
	s.StopServer()
	assert.Len(t, m1, 1)
	assert.Len(t, m2, 1)
	assert.Len(t, s.Clients, 2)
}
