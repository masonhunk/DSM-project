package network

import (
	"io"
	"net"
	"bufio"
	"fmt"
	"encoding/gob"
)

type ITransciever interface {
	Close()
	Send(Message) error
}

type Transciever struct{
	done chan bool
	conn net.Conn
	rw *bufio.ReadWriter
}


func NewTransciever(conn net.Conn, handler func(Message) error) *Transciever{
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	done := make(chan bool, 1)
	go func() {
		done <- true
		for {
			//Do stuff with connections
			var message Message
			dec := gob.NewDecoder(rw)
			err := dec.Decode(&message)
			if err == io.EOF {
				fmt.Println("Reached end of file.")
				done <- true
				return
			} else if err != nil {
				fmt.Println("transciever stopped because of ", err)
				done <- true
				return
			}
			go handler(message)
		}
	}()
	 <- done
	t := new(Transciever)
	t.done = done
	t.conn = conn
	t.rw = rw
	return t
}

func (t *Transciever) Close(){
	t.conn.Close()
	<- t.done
}

func (t *Transciever) Send(message Message) error {
	if message.From == 0{
		fmt.Print("--> server sending ")
		fmt.Printf("%+v\n",message)	}
	enc := gob.NewEncoder(t.rw)
	err := enc.Encode(message)
	t.rw.Flush()
	return err
}

// A transciever mock

type TranscieverMock struct{
	Messages []Message
}

func NewTranscieverMock() *TranscieverMock{
	t := new(TranscieverMock)
	return t
}

func (t *TranscieverMock) Close(){}

func (t *TranscieverMock) Send(message Message) error {
	t.Messages = append(t.Messages, message)
	return nil
}
