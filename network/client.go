package network

import (
	"net"
	"fmt"
)

type Client struct{
	conn net.Conn
	t Transciever
	handler func(Message)
	running bool
}

func NewClient(handler func(Message)) Client{
	return Client{nil, Transciever{}, handler, false}
}

//Connect to some address, which is a string on the form xxx.xxx.xxx.xxx:xxxx with ip and port.
func (c *Client) Connect(address string) (error){
	fmt.Println("Client connecting.")
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Connection failed")
		return err
	}
	fmt.Println("Client conected")
	c.t = NewTransciever(conn, c.handler)
	return nil
}

func (c *Client) Close(){
	c.t.Close()
}

func (c *Client) Send(message Message){
	c.t.Send(message)
}




