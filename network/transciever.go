package network

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
	"net"
)

type ITransciever interface {
	Close()
	Send(Message) error
}

type LoggableTransciever interface {
	ITransciever
	ShouldLog(b bool)
	SetLogFuncOnSend(f func(message Message))
	SetLogFuncOnReceive(f func(message Message))
}

type Transciever struct {
	done chan bool
	conn net.Conn
	rw   *bufio.ReadWriter
	*gob.Encoder
	*gob.Decoder
	shouldLog        bool
	logFuncOnSend    func(message Message)
	logFuncOnReceive func(message Message)
}

func (t *Transciever) ShouldLog(b bool) {
	t.shouldLog = b
}

func (t *Transciever) SetLogFuncOnSend(f func(message Message)) {
	t.logFuncOnSend = f
}

func (t *Transciever) SetLogFuncOnReceive(f func(message Message)) {
	t.logFuncOnReceive = f
}

func NewTransciever(conn net.Conn, handler func(Message) error) *Transciever {
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	t := new(Transciever)
	t.conn = conn
	t.rw = rw
	t.Encoder = gob.NewEncoder(conn)
	t.Decoder = gob.NewDecoder(conn)
	done := make(chan bool, 1)
	go func() {
		done <- true
		for {
			//Do stuff with connections
			var message Message
			err := t.Decode(&message)
			if err == io.EOF {
				done <- true
				return
			} else if err != nil {
				log.Println("transciever stopped because of ", err)
				done <- true
				return
			}
			if t.shouldLog {
				t.logFuncOnReceive(message.(Message))
			}
			go handler(message.(Message))
		}
	}()
	<-done
	t.done = done
	return t
}

func (t *Transciever) Close() {
	t.conn.Close()
	<-t.done
}

func (t *Transciever) Send(message Message) error {
	err := t.Encode(&message)
	t.rw.Flush()
	if err != nil {
		log.Println("Transciever experienced an error: ", err)
	}
	if t.shouldLog {
		t.logFuncOnSend(message.(Message))
	}
	return nil
}

// A transciever mock

type MultiviewTranscieverMock struct {
	Messages []MultiviewMessage
}

func NewMultiviewTranscieverMock() *MultiviewTranscieverMock {
	t := new(MultiviewTranscieverMock)
	return t
}

func (t *MultiviewTranscieverMock) Close() {}

func (t *MultiviewTranscieverMock) Send(message Message) error {

	t.Messages = append(t.Messages, message.(MultiviewMessage))
	return nil
}
