package network

import (
	"net"
	"fmt"
	"bufio"
	"encoding/gob"
	"io"
)

type Client struct{
	conn net.Conn
	rw *bufio.ReadWriter
	handler func(Message)
	running bool
}

func NewClient(handler func(Message)) Client{
	return Client{nil, nil, handler, false}
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
	c.rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	//When we have connected, we will also start handling messages
	c.running = true
	fmt.Println("Client handling messages")
	go c.handleMessages()
	return nil
}

func (c *Client) Close(){
	c.running = false
}

func (c *Client) Send(message Message) error{
	fmt.Println("Sending message.")
	enc := gob.NewEncoder(c.rw)
	err := enc.Encode(message)
	if err != nil{
		fmt.Println("Couldnt encode message")
		return err
	}
	fmt.Println("Flushing")
	c.rw.Flush()
	fmt.Println("Returning")
	return nil
}

func (c *Client) handleMessages() {
	for {
		if c.running == false{
			c.Close()
			break
		}
		var message Message
		dec := gob.NewDecoder(c.rw)
		err := dec.Decode(&message)
		if err == io.EOF{
			fmt.Println("Reached end of file.")
			return
		} else if err != nil {
			fmt.Println("Failed to decode message.")
			return
		}
		c.handler(message)
	}
}




