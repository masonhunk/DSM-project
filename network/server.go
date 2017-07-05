package network

import (
	"net"
	"fmt"
	"os"
)

const (
	connHost = "localhost"
	connPort = "8080"
	connType = "	"
)


// StartServer s
func StartServer() {
	// Listen for incoming connections.
	l, err := net.Listen(connType, connHost+":"+connPort)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + connHost + ":" + connPort)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()

		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequests(conn)
	}
}


// Handles incoming requests.
func handleRequests(conn net.Conn) {
//TODO: add new connections to connList and remove disconnected clients
}