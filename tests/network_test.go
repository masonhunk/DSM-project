package tests

import (
	"testing"
	"DSM-project/network"
	"time"
	"github.com/stretchr/testify/assert"
)

var messages []network.Message

func messageHandler(message network.Message){
	messages = append(messages, message)
}

func TestServerClientConnect(t *testing.T) {
	messages = []network.Message{}
	s,_ := network.NewServer(messageHandler, "5000")
	go s.StartServer()
	time.Sleep(1000000000)
	c := network.NewClient(messageHandler)
	c.Connect("localhost:5000")
	time.Sleep(1000000000)
	c.Send(network.Message{10, "testing", []byte{}})
	time.Sleep(1000000000)
	c.Close()
	time.Sleep(1000000000)
	s.StopServer()
	assert.Len(t, messages ,1)
}

func TestServer2ClientConnect(t *testing.T) {
	messages = []network.Message{}
	s,_ := network.NewServer(messageHandler, "5001")
	go s.StartServer()
	time.Sleep(1000000000)
	c1 := network.NewClient(messageHandler)
	c2 := network.NewClient(messageHandler)
	c1.Connect("localhost:5001")
	c2.Connect("localhost:5001")
	time.Sleep(1000000000)
	c1.Send(network.Message{10, "testing", []byte{}})
	c2.Send(network.Message{11, "testing", []byte{}})
	c1.Send(network.Message{12, "testing", []byte{}})
	time.Sleep(1000000000)
	c1.Close()
	c2.Close()
	time.Sleep(1000000000)
	s.StopServer()
	assert.Len(t, messages ,3)
}