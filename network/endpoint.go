package network

import (
	"net"
	"fmt"
	"bufio"
	"io"
	"encoding/gob"
)

type Message struct{
	Id int //The id of the recipient
	Message string //The message
	Data []byte //Data of the message
}

type Endpoint struct{
	Listener net.Listener
	Handler func(io.ReadWriter, Message)
	Running bool
}

func (m *Message) GetMessage() string{
	return m.Message
}

func NewEndpoint(port string, handler func(io.ReadWriter, Message)) (Endpoint, error) {
	var err error
	e := Endpoint{}
	e.Handler = handler
	e.Listener, err = net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Failed to listen")
		fmt.Println(err)
		return e, err
	}
	return e, nil
}

func (e *Endpoint) Listen() error{
	fmt.Println("EP trying to listen")
	e.Running = true
	for {
		fmt.Println("EP for loop running")
		if e.Running == false{
			fmt.Println("EP stoppped listening")
			e.Listener.Close()
			break
		}
		fmt.Println("EP waiting for accept")
		conn, err := e.Listener.Accept()
		fmt.Print("EP got accept")
		if err != nil{
			fmt.Println("Failed to accept connection.")
			return err
		}
		fmt.Println("EP handles message")
		go e.handleMessages(conn)
	}
	return nil
}

func (e *Endpoint) Close() {
	e.Running = false
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
		fmt.Println("EP got a message")
		e.Handler(rw, message)
	}
}


