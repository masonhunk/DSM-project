package network

import (
	"net"
	"fmt"
)

type IClient interface {
	Connect(address string) error
	Close()
	Send(message Message) error
	GetTransciever() ITransciever
}

type Client struct{
	conn net.Conn
	t ITransciever
	handler func(Message) error
	running bool
}

func (c *Client) GetTransciever() ITransciever {
	return c.t
}

func NewClient(handler func(Message) error) *Client{
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
	c.GetTransciever().Close()
}

func (c *Client) Send(message Message) error {
	return c.GetTransciever().Send(message)
}




