package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Connection interface {
	Connect(ip string, port int) (int, error)
	Close()
}

var _ Connection = new(connection)

type connection struct {
	myId     int
	myPort   int
	running  bool
	group    *sync.WaitGroup
	listener *net.TCPListener
	peers    []*peer
	in, out  chan []byte
}

type peer struct {
	id   int
	ip   string
	port int
	conn *net.TCPConn
}

/*
	NewConnection gives a new connection, that will be listening on the given port.
	The host will have ID 0 at this point.
*/
func NewConnection(port int, bufferSize int) (*connection, <-chan []byte, chan<- []byte) {
	c := new(connection)
	c.peers = make([]*peer, 1)
	c.in, c.out = make(chan []byte, 1000), make(chan []byte, 1000)
	c.running = true
	c.group = new(sync.WaitGroup)
	c.myPort = port
	listener, err := net.Listen("tcp", fmt.Sprint(":", port))
	if err != nil {
		panic("Couldnt create listener: " + err.Error())
	}
	c.listener = listener.(*net.TCPListener)
	go c.listen()
	go c.sendLoop()
	return c, c.in, c.out
}

/*
	This function connects the host to another host on the given ip and port.
	When connected, this host will receive a new ID from the host.
	This will also make this host start listening and sending messages, which will then be passed through the channels
	given when the connection was initialized.
*/
func (c *connection) Connect(ip string, port int) (int, error) {
	var err error
	var tempConn net.Conn
	tempConn, err = net.DialTimeout("tcp", fmt.Sprint(ip, ":", port), time.Second*5)
	for err != nil {
		tempConn, err = net.DialTimeout("tcp", fmt.Sprint(ip, ":", port), time.Second*5)
	}
	conn := tempConn.(*net.TCPConn)
	if err != nil {
		panic("Couldnt dial the host: " + err.Error())
	}

	write(conn, []byte{0, 0, byte(c.myPort / 256), byte(c.myPort % 256)})
	//conn.SetReadDeadline(time.Now().Add(time.Second*5))

	msg := read(conn)
	c.myId = int(msg[0])
	c.addPeer(c.myId, nil, 0)
	otherId := int(msg[1])
	c.peers = make([]*peer, c.myId+1)
	c.addPeer(otherId, conn, port)
	j := 2
	for j < len(msg) {
		id := int(msg[j])
		j++
		ip, port, k := addrFromBytes(msg[j:])
		newConn, err := net.Dial("tcp", fmt.Sprint(ip, ":", port))
		if err != nil {
			panic("Got error when connecting to addr " + fmt.Sprint(ip, ":", port) + ": " + err.Error())
		}
		write(newConn, []byte{0, byte(c.myId), byte(c.myPort / 256), byte(c.myPort % 256)})
		c.addPeer(id, newConn.(*net.TCPConn), port)
		go c.receive(c.peers[id])
		j += k
	}
	go c.receive(c.peers[otherId])
	return c.myId, nil
}

func (c *connection) Close() {
	close(c.out)
	c.group.Wait()

}

/*
	This is a locally used method that sends messages read from the incoming channel.
*/
func (c *connection) sendLoop() {
	c.group.Add(1)
	var id int
	for msg := range c.out {
		time.Sleep(0)
		id = int(msg[0])

		if id == c.myId {
			c.in <- msg
		} else {
			if id >= len(c.peers) {
				go func() {
					time.Sleep(time.Millisecond * 500)
					c.out <- msg
				}()
			} else {
				msg[0] = 1

				write(c.peers[id].conn, msg)
			}
		}
	}
	c.running = false
	c.group.Done()
	c.group.Wait()
	close(c.in)
}

/*
	This is a locally used method that reads messages from a certain connection, and puts them in the
	outgoing channel.
*/
func (c *connection) receive(peer *peer) {
	c.group.Add(1)
	for c.running {

		b := read(peer.conn)
		if b == nil {
			continue
		}
		if peer.id == 0 || peer.id == 1 {
		}

		if len(b) > 0 && b[0] == 0 {
			id := int(b[1])
			ip, port, _ := addrFromBytes(b[2:])
			conn := c.connectToHost(ip, port)
			c.addPeer(id, conn, port)
		} else {
			msg := append([]byte{byte(peer.id)}, b[1:]...)
			c.in <- msg

		}
	}
	peer.conn.Close()
	c.group.Done()
}

/*
	Local functions that just runs a loop listening for incoming connections and running addHost
	on all new connections.
*/
func (c *connection) listen() {
	c.group.Add(1)

	for c.running {
		c.listener.SetDeadline(time.Now().Add(time.Millisecond * 500))
		conn, err := c.listener.AcceptTCP()
		if err == nil {
			c.addHost(conn)
		} else if !strings.HasSuffix(err.Error(), "i/o timeout") {
			panic("Something crashed when we were accepting: " + err.Error())
		}
	}
	c.listener.Close()
	c.group.Done()
}

/*
	If the host sends a message with ID = 0, it is a new host joining the network, so we send it our list of peers.
	If the ID is different from 0, the host has already joined the network, and should just be added to our list of peers.
*/
func (c *connection) addHost(conn *net.TCPConn) {
	msg := read(conn)
	var buf bytes.Buffer
	id := int(msg[1])
	port := int(msg[2])*256 + int(msg[3])
	if id == 0 {
		id = len(c.peers)
		buf.Write([]byte{byte(id), byte(c.myId)})
		for i := range c.peers {
			if i != c.myId {
				buf.WriteByte(byte(i))
				buf.Write(addrToBytes(c.peers[i].ip, c.peers[i].port))
			}
		}
		write(conn, buf.Bytes())

	}
	c.addPeer(id, conn, port)
	go c.receive(c.peers[id])
}

/*
	This functions connects to a host with the given address and returns the TCP connection pointer.
*/
func (c *connection) connectToHost(ip string, port int) *net.TCPConn {
	var conn net.Conn
	var err error
	for c.running {
		conn, err = net.DialTimeout("tcp", fmt.Sprint(ip, ":", port), time.Millisecond*500)
		if err != nil && !strings.HasSuffix(err.Error(), "i/o timeout") {
			panic("Something went wrong when trying to connect to " + fmt.Sprint(ip, ":", port))
		}
	}
	return conn.(*net.TCPConn)
}

func (c *connection) addPeer(id int, conn *net.TCPConn, port int) {
	if len(c.peers) <= id {
		c.peers = append(c.peers, make([]*peer, (id-len(c.peers))+1)...)
	}
	ip := "localhost"
	if conn != nil {
		addr := conn.RemoteAddr().String()
		splitAddr := strings.Split(addr, ":")
		if len(splitAddr) == 2 {
			ip = splitAddr[0]
		}
	}

	c.peers[id] = &peer{id, ip, port, conn}
}

func (c *connection) getAddr(id int) string {
	peer := c.peers[id]
	return fmt.Sprint(peer.ip, ":", peer.port)
}

func write(conn net.Conn, data []byte) {
	length := uint64(len(data))
	if len(data) != int(length) {
		panic(fmt.Sprint("Length did not match.", length, len(data)))
	}
	l := make([]byte, 8)
	binary.PutVarint(l, int64(len(data)))
	msg := append(l, data...)
	n := 0
	var err error
	for {
		n, err = conn.Write(msg[n:])
		if err == nil {
			break
		}
		panic(err.Error())
	}
}

func read(conn net.Conn) []byte {
	length := make([]byte, 8)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, err := io.ReadFull(conn, length)
	if err != nil {
		return nil
	}
	l, _ := binary.Varint(length)
	msg := make([]byte, l)
	_, err = io.ReadFull(conn, msg)
	if err != nil {
		return nil
	}
	return msg
}

func addrFromBytes(b []byte) (string, int, int) {
	s := make([]string, 4)
	for i := 0; i < 4; i++ {
		s[i] = strconv.Itoa(int(b[i]))
	}
	ip := strings.Join(s, ".")
	port, i := binary.Varint(b[4:])
	return ip, int(port), i + 4
}

func addrToBytes(ip string, port int) []byte {
	ipArray := strings.Split(ip, ".")
	if len(ipArray) != 4 {
		ipArray = []string{"127", "0", "0", "1"}
	}
	ip_in_bytes := make([]byte, 4)
	for i := range ipArray {
		v, _ := strconv.Atoi(ipArray[i])
		ip_in_bytes[i] = byte(v)
	}
	buf := make([]byte, binary.Size(int64(port)))
	i := binary.PutVarint(buf, int64(port))
	result := append(ip_in_bytes, buf[:i]...)
	return result
}
