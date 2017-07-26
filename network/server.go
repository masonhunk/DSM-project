package network

import (
	"log"
	"net"
)

type Server struct {
	port    string
	Clients map[byte]*Transciever
	nonce   byte
	ep      Endpoint
	handler func(Message) error
	logger 	CSVStructLogger
}

func NewServer(handler func(Message) error, port string, logger CSVStructLogger) (Server, error) {
	s := Server{port, make(map[byte]*Transciever), byte(0), Endpoint{}, handler, logger}
	var err error
	s.ep, err = NewEndpoint(port, s.handleConnection)
	if err != nil {
		log.Println("Could not create endpoint:", err)
		return Server{}, err
	}
	return s, nil
}

func (s *Server) StopServer() {
	for _, v := range s.Clients {
		v.Close()
	}
	defer s.logger.Close()
	defer s.ep.Close()
}

func (s *Server) Send(message Message) {
	t := s.Clients[message.GetTo()]
	t.Send(message)
}

func (s *Server) handleMessage(message Message) error {
	if message.GetTo() != byte(255) {
		s.Send(message)
	}
	if &s.logger != nil {
		s.logger.Log(message, MessageEncoder{})
	}
	return s.handler(message)
}

func (s *Server) handleConnection(conn net.Conn) {
	t := NewTransciever(conn, s.handleMessage)
	t.Send(SimpleMessage{From: 255, To: s.nonce, Type: "WELC"})
	s.Clients[s.nonce] = t
	s.nonce += byte(1)
}
