package network

import (
	"net"
	"fmt"
)

type IClient interface {
	Connect(address string) error
	Close()
	Send(message Message) error
}

type Client struct{
	conn net.Conn
	t *Transciever
	handler func(Message)
	running bool
}

func NewClient(handler func(Message)) *Client{
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
		fmt.Println("Connection failed")
		return err
	}
	c.t = NewTransciever(conn, c.handler)
	return nil
}

func (c *Client) Close(){
	c.t.Close()
}

func (c *Client) Send(message Message) error {
	return c.t.Send(message)
}




