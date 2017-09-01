package network

import (
	"DSM-project/utils"
	"bytes"
	"errors"
	"fmt"
	"github.com/davecgh/go-xdr/xdr2"
	"log"
	"net"
	"sync"
	"time"
)

type IClient interface {
	Connect(address string) error
	Close()
	Send(message Message) error
	GetTransciever() ITransciever
}

type P2PClient struct {
	conn     Connection
	handler  func(Message) error
	running  bool
	in       <-chan []byte
	out      chan<- []byte
	shutdown chan bool
	group    *sync.WaitGroup
}

func NewP2PClient(handler func(Message) error) *P2PClient {
	c := new(P2PClient)
	c.handler = handler
	c.group = new(sync.WaitGroup)
	c.shutdown = make(chan bool)
	return c
}

func (c *P2PClient) Connect(address string) error {
	ip, port := utils.StringToIpAndPort(address)
	err := errors.New("Not connected yet.")
	for i := 1; i < 20 && err != nil; i++ {
		c.conn, c.in, c.out, err = NewConnection(port+i, 1000)
	}
	if err != nil {
		panic(fmt.Sprint("Couldn't start listener. Tried ports ", port+1, " - ", port+19, ".\n"+
			"Error was ", err.Error()))
	}
	myId, _ := c.conn.Connect(ip, port)
	go c.recieveLoop()
	welcomeMsg := SimpleMessage{From: 255, To: byte(myId), Type: "WELC"}
	go c.handler(welcomeMsg)
	return nil
}

func (c *P2PClient) recieveLoop() {
	c.group.Add(1)
	buf := bytes.NewBuffer([]byte{})
Loop:
	for {
		time.Sleep(0)
		var data []byte
		select {
		case data = <-c.in:
		case <-c.shutdown:
			break Loop
		}
		buf.Write(data[1:])
		var multiviewMsg MultiviewMessage
		_, err := xdr.Unmarshal(buf, &multiviewMsg)
		if err != nil {
			panic(err.Error())
		}
		go c.handler(multiviewMsg)
	}
	c.group.Done()
}

func (c *P2PClient) Close() {
	c.shutdown <- true
	c.group.Wait()
	c.conn.Close()
}

func (c *P2PClient) Send(message Message) error {
	msg := message.(MultiviewMessage)
	var w bytes.Buffer
	_, err := xdr.Marshal(&w, &msg)
	if err != nil {
		panic("Error: " + err.Error())
	}
	data := make([]byte, w.Len()+1)
	data[0] = msg.GetTo()

	w.Read(data[1:])
	c.out <- data
	return nil
}

func (c *P2PClient) GetTransciever() ITransciever {
	return c
}

type Client struct {
	conn    net.Conn
	t       ITransciever
	handler func(Message) error
	running bool
}

func (c *Client) GetTransciever() ITransciever {
	return c.t
}

func NewClient(handler func(Message) error) *Client {
	c := new(Client)
	c.conn = nil
	c.t = &Transciever{}
	c.handler = handler
	c.running = false
	return c
}

//Connect to some address, which is a string on the form xxx.xxx.xxx.xxx:xxxx with ip and port.
func (c *Client) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Println("Connection failed:", err)
		return err
	}
	c.t = NewTransciever(conn, c.handler)
	return nil
}

func (c *Client) Close() {
	c.GetTransciever().Close()
}

func (c *Client) Send(message Message) error {
	return c.GetTransciever().Send(message)
}
