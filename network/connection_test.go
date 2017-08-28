package network

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
)

func TestConnection_iptransform(t *testing.T) {
	ip := []string{
		"192.168.1.12",
		"192.168.1.11",
		"192.168.1.255",
		"localhost",
		"[::1]",
	}
	ipExpected := []string{
		"192.168.1.12",
		"192.168.1.11",
		"192.168.1.255",
		"127.0.0.1",
		"127.0.0.1",
	}
	port := []int{
		2255,
		21245,
		22,
		11,
		22,
	}
	var sumaddrb []byte
	for i := range ip{
		addrb := addrToBytes(ip[i], port[i])
		sumaddrb = append(sumaddrb, addrb...)
		ipc, portc, _ := addrFromBytes(addrb)
		assert.Equal(t, ipExpected[i], ipc)
		assert.Equal(t, port[i], portc)
	}
	ipclist := make([]string, 0)
	portclist := make([]int, 0)
	i := 0;
	for  i < len(sumaddrb){
		ipc, portc, k := addrFromBytes(sumaddrb[i:])
		ipclist = append(ipclist, ipc)
		portclist = append(portclist, portc)
		i += k
	}
	assert.Equal(t, ipExpected, ipclist)
	assert.Equal(t, port, portclist)
}

func TestNewConnection(t *testing.T) {
	c0,_,_ := NewConnection(2334, 10)
	c1,_,_ := NewConnection(1123, 10)
	c2,_,_ := NewConnection(1523, 10)
	control1 := make(chan bool)
	control2 := make(chan bool)
	assert.True(t, c0.running)
	assert.Len(t, c0.peers, 1)
	assert.Equal(t, 0, c0.myId)
	go func(){
		<- control1
		myId, err := c1.Connect("localhost", 2334)
		assert.Nil(t, err)
		assert.Equal(t, 1, myId)
		control1 <- true

	}()
	go func(){
		<-control2
		myId, err := c2.Connect("localhost", 2334)
		assert.Nil(t, err)
		assert.Equal(t, 2, myId)
		control2 <- true

	}()
	control1 <- true
	assert.True(t, <- control1)
	time.Sleep(time.Millisecond*500)
	assert.Equal(t, c0.myId, 0)
	assert.Equal(t, c1.myId, 1)
	assert.Len(t, c0.peers, 2)
	assert.Len(t, c1.peers, 2)
	assert.Len(t, c2.peers, 1)
	control2 <- true
	assert.True(t, <- control2)
	time.Sleep(time.Millisecond*500)
	assert.Equal(t, c0.myId, 0)
	assert.Equal(t, c1.myId, 1)
	assert.Equal(t, c2.myId, 2)
	assert.Len(t, c0.peers, 3)
	assert.Len(t, c1.peers, 3)
	assert.Len(t, c2.peers, 3)
	c0.out <- []byte{0, 0, 1, 2, 3}
	c0.out <- []byte{1, 0, 1, 2, 3}
	c0.out <- []byte{2, 0, 1, 2, 3}
	c1.out <- []byte{0, 1, 2, 3, 4}
	c1.out <- []byte{1, 1, 2, 3, 4}
	c1.out <- []byte{2, 1, 2, 3, 4}
	c2.out <- []byte{0, 2, 3, 4, 5}
	c2.out <- []byte{1, 2, 3, 4, 5}
	c2.out <- []byte{2, 2, 3, 4, 5}
	c0expected := [][]byte{
		{0, 0, 1, 2, 3},
		{1, 1, 2, 3, 4},
		{2, 2, 3, 4, 5},
	}
	c1expected := [][]byte{
		{0, 0, 1, 2, 3},
		{1, 1, 2, 3, 4},
		{2, 2, 3, 4, 5},
	}
	c2expected := [][]byte{
		{0, 0, 1, 2, 3},
		{1, 1, 2, 3, 4},
		{2, 2, 3, 4, 5},
	}

	c0Rec := make([][]byte, 0)
	c1Rec := make([][]byte, 0)
	c2Rec := make([][]byte, 0)
	looping := true
	for looping {
		select {
		case msg := <- c0.in:
			c0Rec = append(c0Rec, msg)
		case msg := <- c1.in:
			c1Rec = append(c1Rec, msg)
		case msg := <- c2.in:
			c2Rec = append(c2Rec, msg)
		case <- time.After(time.Second):
			looping = false
		}
	}
	assert.Len(t, c0Rec, 3)
	assert.Len(t, c1Rec, 3)
	assert.Len(t, c2Rec, 3)
	for i := range c0Rec {
		assert.Contains(t, c0expected, c0Rec[i])
	}
	for i := range c1Rec {
		assert.Contains(t, c1expected, c1Rec[i])
	}
	for i := range c2Rec {
		assert.Contains(t, c2expected, c2Rec[i])
	}

}

