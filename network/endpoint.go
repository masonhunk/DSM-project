package network

import (
	"net"
	"fmt"
)

type SimpleMessage struct{
	From byte
	To byte
	Type string
}

func (m SimpleMessage) GetFrom() byte {
	return m.From
}

func (m SimpleMessage) GetTo() byte {
	return m.To
}

func (m SimpleMessage) GetType() string {
	return m.Type
}

type MultiviewMessage struct{
	From byte
	To byte
	Type string
	Fault_addr int
	Minipage_size int
	Minipage_base int // addrress in the vpage address space
	Privbase int //address in the privileged view
	EventId byte
	Err error
	Data []byte //Data of the message
}

func (m MultiviewMessage) GetType() string {
	return m.Type
}

func (m MultiviewMessage) GetFrom() byte {
	return m.From
}

func (m MultiviewMessage) GetTo() byte {
	return m.To
}
type Message interface {
	GetFrom() byte
	GetTo() byte
	GetType() string
}

type Endpoint struct{
	done chan bool
	l net.Listener
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
				fmt.Println("Endpoint - Accept failed. ", err)
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



