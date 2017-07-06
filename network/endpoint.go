package network

import (
	"net"
	"fmt"
	"bufio"
	"io"
	"encoding/gob"
)

type Message struct{
	id string //The id of the recipient
	message string //The message
	data []byte //Data of the message
}

type Endpoint struct{
	listener net.Listener
	handler func(message Message)
}

func (e *Endpoint) listen(port string) error{
	var err error
	e.listener, err = net.Listen("tcp", port)
	if err != nil{
		fmt.Println("Failed to liste.")
		return err
	}
	for {
		conn, err := e.listener.Accept()
		if err != nil{
			fmt.Println("Failed to accept connection.")
			return err
		}
		go e.handleMessages(conn)

	}
}

func (e *Endpoint) handleMessages(conn net.Conn) {
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer conn.Close()
	for {
		var message Message
		dec := gob.NewDecoder(rw)
		err := dec.Decode(&message)
		if err == io.EOF{
			fmt.Println("Reached end of file.")
			return
		} else if err != nil {
			fmt.Println("Failed to decode message.")
			return
		}
		e.handler(message)
	}
}


