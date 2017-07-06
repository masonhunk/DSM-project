package network

import (
	"io"
	"fmt"
)

type Server struct{
	port string
	clients map[int]io.ReadWriter
	nonce int
	ep Endpoint
	handler func(Message)
}

func NewServer(handler func(Message), port string) (Server, error){
	s := Server{port, make(map[int]io.ReadWriter), 1, Endpoint{}, handler}
	var err error
	s.ep, err = NewEndpoint(port, s.handleMessage)
	if err != nil {
		fmt.Println("Could not create endpoint")
		fmt.Println(err)
		return Server{}, err
	}

	return s, nil
}

// StartServer s
func (s *Server) StartServer() {
	//We first create an endpoint, give it a handler function and start listening for messages
	fmt.Println("Telling ep to listen")
	s.ep.Listen()
}

func (s *Server) StopServer() {
	s.ep.Close()
}


// Handles incoming requests.
func (s *Server)handleMessage(rw io.ReadWriter, message Message) {
	if message.Message == "join"{
		s.clients[s.nonce] = rw
		s.nonce = s.nonce+1
	}
	s.handler(message)
}