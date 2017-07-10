package network

import (
	"fmt"
	"net"
)

type Server struct{
	port string
	Clients map[byte]Transciever
	nonce byte
	ep Endpoint
	handler func(Message)
}

func NewServer(handler func(Message), port string) (Server, error){
	s := Server{port, make(map[byte]Transciever), byte(1), Endpoint{}, handler}
	var err error
	s.ep, err = NewEndpoint(port, s.handleConnection)
	if err != nil {
		fmt.Println("Could not create endpoint")
		fmt.Println(err)
		return Server{}, err
	}

	return s, nil
}



func (s *Server) StopServer() {
	for _,v := range s.Clients{
		v.Close()
	}
	s.ep.Close()
}

func (s *Server) Send(message Message) {
	t := s.Clients[message.To]
	t.Send(message)
}

func (s *Server) handleMessage(message Message) {
	if message.To != 0 {
		s.Send(message)
	}
	s.handler(message)
}


func (s *Server)handleConnection(conn net.Conn) {
	t := NewTransciever(conn, s.handleMessage)
	s.Clients[s.nonce] = t
	s.nonce = s.nonce + byte(1)
}