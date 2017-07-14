package network

import (
	"io"
	"net"
	"bufio"
	"encoding/gob"
	"time"
	"log"
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
				done <- true
				return
			} else if err != nil {
				log.Println("transciever stopped because of ", err)
				done <- true
				return
			}
			log.Println("Transciever recieved message : ", message)
			go handler(message.(Message))
			time.Sleep(time.Millisecond*100)
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
	if message.GetFrom() == 0{
		log.Print("--> server sending ")
		log.Printf("%+v\n",message)	}
	enc := gob.NewEncoder(t.rw)
	err := enc.Encode(&message)
	t.rw.Flush()
	if err != nil {
		log.Println("Transciever experienced an error: ", err)
	}
	return err
}

// A transciever mock

type MultiviewTranscieverMock struct{
	Messages []MultiviewMessage
}

func NewMultiviewTranscieverMock() *MultiviewTranscieverMock{
	t := new(MultiviewTranscieverMock)
	return t
}

func (t *MultiviewTranscieverMock) Close(){}

func (t *MultiviewTranscieverMock) Send(message Message) error {

	t.Messages = append(t.Messages, message.(MultiviewMessage))
	return nil
}
