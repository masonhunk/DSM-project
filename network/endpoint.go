package network

import (
	"net"
	"fmt"
)

type Message struct{
	Id int //The id of the recipient
	Message string //The message
	Data []byte //Data of the message
}

type Endpoint struct{
	done chan bool
	l net.Listener
}


func (m *Message) GetMessage() string{
	return m.Message
}

func NewEndpoint(port string, handler func(conn net.Conn)) (Endpoint, error) {
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Failed to listen")
		fmt.Println(err)
		return Endpoint{}, err
	}
	done := make(chan bool)
	running := make(chan bool)
	go func() {
		running <- true
		for {
			//Do stuff with connections
			conn, err := l.Accept()
			if err != nil{
				fmt.Println("Endpoint - Accept failed.")
				l.Close()
				done <- true
				return

			}
			handler(conn)

		}
	}()
	 <- running
	return Endpoint{done, l}, nil
}


func (e *Endpoint) Close() {
	e.l.Close()
	<- e.done
}



