package Benchmarks

import (
	"DSM-project/dsm-api/treadmarks"
	"fmt"
	"io"
	"log"
	"runtime/pprof"
	"time"
)

func TestBarrierTimeTM(nrTimes, nrhosts int, pprofFile io.Writer) {
	tm1, tms := setupTMHosts(nrhosts, 4096, 4096)

	var startTime time.Time
	startTime = time.Now()
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	for i := 0; i < nrTimes; i++ {
		for _, mw := range tms {
			go mw.Barrier(0)
		}
		tm1.Barrier(0)
	}
	pprof.StopCPUProfile()

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	for _, tm := range tms {
		tm.Shutdown()
	}
	tm1.Shutdown()
}

func TestLockTM(nrTimes int, pprofFile io.Writer) {
	tm1, tms := setupTMHosts(8, 4096, 4096)

	var startTime time.Time
	startTime = time.Now()
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	for i := 0; i < nrTimes; i++ {
		//fmt.Println("about to acquire lock at host", i%7)
		tms[i%7].AcquireLock(0)
		//fmt.Println("acquired lock at host", i%7)
		tms[i%7].ReleaseLock(0)
		//fmt.Println("released lock at host", i%7)

	}
	pprof.StopCPUProfile()

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	for _, tm := range tms {
		tm.Shutdown()
	}
	tm1.Shutdown()
}

func TestSynchronizedReadsWritesTM(nrRounds int, pprofFile io.Writer) {
	tm1, tms := setupTMHosts(2, 4096, 4096)
	addr, _ := tm1.Malloc(1)

	var startTime time.Time
	startTime = time.Now()
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}

	for i := 0; i < nrRounds; i++ {
		tm1.AcquireLock(0)
		tm1.Write(addr, byte(1))
		tm1.ReleaseLock(0)
		tms[0].AcquireLock(0)
		tms[0].Read(addr)
		tms[0].ReleaseLock(0)
	}
	pprof.StopCPUProfile()

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	for _, tm := range tms {
		tm.Shutdown()
	}
	tm1.Shutdown()
}

func TestNonSynchronizedReadWritesTM(nrRounds int, pprofFile io.Writer) {
	tm1, _ := setupTMHosts(1, 4096, 4096)
	addr, _ := tm1.Malloc(1)
	tm1.Write(addr, byte(1))
	var startTime time.Time
	startTime = time.Now()
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}

	for i := 0; i < nrRounds; i++ {
		tm1.Write(addr, byte(1))
		tm1.Read(addr)
	}

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	tm1.Shutdown()
}

func readOnAllTMHosts(addr int, tms []*treadmarks.TreadmarksApi) {
	for _, tm := range tms {
		tm.Read(addr)
	}
}

func setupTMHosts(nrHosts int, memSize, pageByteSize int) (manager *treadmarks.TreadmarksApi, mws []*treadmarks.TreadmarksApi) {
	manager, _ = treadmarks.NewTreadmarksApi(memSize, pageByteSize, uint8(nrHosts), uint8(nrHosts), uint8(nrHosts))
	manager.Initialize(2000)
	mws = make([]*treadmarks.TreadmarksApi, nrHosts-1)
	for i := range mws {
		mws[i], _ = treadmarks.NewTreadmarksApi(memSize, pageByteSize, uint8(nrHosts), uint8(nrHosts), uint8(nrHosts))
		mws[i].Initialize(2000 + i + 1)
		mws[i].Join("localhost", 2000)
	}
	return
}
