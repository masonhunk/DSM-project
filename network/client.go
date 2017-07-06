package network

import (
	"net"
	"fmt"
	"os"
	"bufio"
	"errors"
	"go/types"
	"encoding/gob"
)

type Client struct{
	conn net.Conn
	rw *bufio.ReadWriter
	h func(Message)
}


//Connect to some address, which is a string on the form xxx.xxx.xxx.xxx:xxxx with ip and port.
func (c *Client) Connect(address string) (error){
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	c.rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	//When we have connected, we will also start handling messages
	return nil
}

func (c *Client) Close(){
	c.conn.Close()
}

func (c *Client) Send(message Message) error{
	enc := gob.NewEncoder(c.rw)
	err := enc.Encode(message)
	if err != nil{
		fmt.Println("Couldnt encode message")
		return err
	}
	c.rw.Flush()
	return nil
}




