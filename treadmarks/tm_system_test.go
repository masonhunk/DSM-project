package treadmarks

import (
	"DSM-project/memory"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func setupTreadMarksStruct(nrProcs int) TreadMarks {
	vm1 := memory.NewVmem(128, 8)
	tm1 := NewTreadMarks(vm1, nrProcs, 4, 4)
	return *tm1
}

func InitialiseTMSystem(nrProcs int) (tm *TreadMarks, tms []*TreadMarks) {

	waitchan := make(chan string)
	for i := 1; i <= nrProcs; i++ {
		t := setupTreadMarksStruct(nrProcs)
		if i == 1 {
			go func(tm TreadMarks) {
				if err := tm.Startup(); err != nil {
					log.Println("failed to start up tm instance with error:", err)
					waitchan <- "err"
					return
				}
				tm = t
				waitchan <- "ok"
			}(t)

		} else {
			go func(tm TreadMarks) {
				if err := tm.Join("localhost:2000"); err != nil {
					log.Println("failed to join with error:", err)
					waitchan <- "err"
					return
				}
				tms = append(tms, &t)
				waitchan <- "ok"
			}(t)
		}
		if <-waitchan == "err" {
			return nil, nil
		}
	}
	return
}

func TestTreadMarksInitialisation(t *testing.T) {
	managerHost, hosts := InitialiseTMSystem(4)
	assert.Equal(t, byte(1), managerHost.ProcId)
	assert.Equal(t, byte(2), hosts[1].ProcId)
}
