package network

import (
	"net"
	"fmt"
)

type Message struct{
	From byte
	To byte
	Type string
	Fault_addr int
	Minipage_size int
	Minipage_base int // addrress in the vpage address space
	Privbase int //address in the privileged view
	Event chan string
	Err error
	Data []byte //Data of the message

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
				fmt.Println(err)
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



