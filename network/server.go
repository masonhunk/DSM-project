package network

import (
	"bytes"
	"github.com/davecgh/go-xdr/xdr2"
	"github.com/orcaman/concurrent-map"
	"log"
	"net"
	"sync"
	"time"
)

type ServerInterface interface {
	Close()
	Send(message Message) error
}

type P2PServer struct {
	conn     Connection
	port     int
	in       <-chan []byte
	out      chan<- []byte
	shutdown chan bool
	group    *sync.WaitGroup
	handler  func(message Message) error
}

func NewP2PServer(handler func(Message) error, port int, logger *CSVStructLogger) (*P2PServer, error) {
	s := new(P2PServer)
	s.conn, s.in, s.out, _ = NewConnection(port, 1000)
	s.shutdown = make(chan bool)
	s.handler = handler
	s.group = new(sync.WaitGroup)
	go s.recieveLoop()
	return s, nil
}

func (s *P2PServer) Close() {
	s.conn.Close()
	s.shutdown <- true
	s.group.Wait()
}

func (s *P2PServer) Send(message Message) error {
	msg := message.(MultiviewMessage)
	var w bytes.Buffer
	_, err := xdr.Marshal(&w, &msg)
	if err != nil {
		panic("Error: " + err.Error())
	}
	data := make([]byte, w.Len()+1)
	data[0] = msg.GetTo()

	w.Read(data[1:])
	s.out <- data
	return nil
}

func (s *P2PServer) recieveLoop() {
	s.group.Add(1)
	buf := bytes.NewBuffer([]byte{})
Loop:
	for {
		time.Sleep(0)
		var data []byte
		select {
		case data = <-s.in:
		case <-s.shutdown:
			break Loop
		}
		buf.Write(data[1:])
		var multiviewMsg MultiviewMessage
		_, err := xdr.Unmarshal(buf, &multiviewMsg)
		if err != nil {
			panic(err.Error())
		}
		go s.handler(multiviewMsg)
	}
	s.group.Done()
}

type Server struct {
	port    int
	Clients cmap.ConcurrentMap
	//Clients map[byte]*Transciever
	nonce   byte
	ep      Endpoint
	handler func(Message) error
	logger  *CSVStructLogger
}

func NewServer(handler func(Message) error, port int, logger *CSVStructLogger) (*Server, error) {
	s := &Server{port, cmap.New(), byte(0), Endpoint{}, handler, logger}
	var err error
	s.ep, err = NewEndpoint(port, s.handleConnection)
	if err != nil {
		log.Println("Could not create endpoint:", err)
		return nil, err
	}
	return s, nil
}

func (s *Server) Close() {
	for _, v := range s.Clients.Items() {
		(v.(*Transciever)).Close()
	}
	defer s.logger.Close()
	defer s.ep.Close()
}

func (s *Server) Send(message Message) error {
	t, ok := s.Clients.Get(string(message.GetTo()))
	if ok {
		t.(*Transciever).Send(message)
	} else {
		panic("unable to send message at server")
	}
	return nil
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
	s.Clients.Set(string(s.nonce), t)
	s.nonce += byte(1)
}
