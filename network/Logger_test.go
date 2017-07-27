package network

import (
	"testing"
	"os"
	"log"
	"time"
)

func TestNewCSVLogger(t *testing.T) {
	f, err := os.Create("test.csv")
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}

	l := NewCSVStructLogger(f)
	msg := SimpleMessage{
		0,
		1,
		"test",
	}
	l.Log(msg, &MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	l.Log(msg, MessageEncoder{})
	time.Sleep(time.Millisecond * 500)
	l.Close()
}