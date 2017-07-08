package network

import (
	"io"
	"net"
	"bufio"
	"fmt"
	"encoding/gob"
	"reflect"
)

type Transciever struct{
	done chan bool
	conn net.Conn
	rw *bufio.ReadWriter
}

func NewTransciever(conn net.Conn, handler func(Message)) Transciever{
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	done := make(chan bool)
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
				fmt.Println("transciever stopped.")
				fmt.Println(reflect.TypeOf(err))
				done <- true
				return
			}
			handler(message)
		}
	}()
	 <- done
	return Transciever{done, conn, rw}
}

func (t *Transciever) Close(){
	t.conn.Close()
	<- t.done
}

func (t *Transciever) Send(message Message) {
	fmt.Println("Sending message.")
	enc := gob.NewEncoder(t.rw)
	err := enc.Encode(message)
	if err != nil{
		fmt.Println("Couldnt encode message")
	}
	t.rw.Flush()
	fmt.Println("Flushing")
	fmt.Println("Returning")
}
