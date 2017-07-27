package network

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"time"
)

type CSVLogger interface {
	Log(s interface{})
	Close()
}

type StructEncoder interface {
	GetHeaders(s interface{}) []string
	GetValues(s interface{}) []string
}

type CSVStructLogger struct {
	*csv.Writer
	consumerChan chan []string
}

type MessageEncoder struct {}

func (MessageEncoder) GetHeaders(s interface{}) []string {
	return []string{"From", "To", "Type"}
}

func (MessageEncoder) GetValues(s interface{}) []string {
	msg := s.(Message)
	return []string{fmt.Sprint(msg.GetFrom()), fmt.Sprint(msg.GetTo()), msg.GetType()}
}

func NewCSVStructLogger(writer io.Writer) *CSVStructLogger {
	res := CSVStructLogger{
		Writer: csv.NewWriter(writer),
		consumerChan: make(chan []string, 30),
	}
	go func(c *chan []string) {
		for data := range *c {
			if err := res.Write(data); err != nil {
				log.Fatal(data)
			}
		}
		defer res.Flush()
	}(&res.consumerChan)
	return &res
}

func (l *CSVStructLogger) Log(s interface{}, encoder StructEncoder) {
	res := append(encoder.GetValues(s), time.Now().Format(time.StampMilli))
	l.consumerChan <- res
}

func (l *CSVStructLogger) Close() {
	close(l.consumerChan)
}
