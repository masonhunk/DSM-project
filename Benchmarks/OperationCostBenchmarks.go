package Benchmarks

import (
	"DSM-project/multiview"
	"fmt"
	"io"
	"log"
	"runtime/pprof"
	"time"
)

func TestBarrierTimeMW(nrTimes, nrhosts int, pprofFile io.Writer) {
	mw1, mws := setupHosts(nrhosts, 4096, 4096)

	var startTime time.Time
	startTime = time.Now()
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	for i := 0; i < nrTimes; i++ {
		for _, mw := range mws {
			go mw.Barrier(0)
		}
		mw1.Barrier(0)
	}
	pprof.StopCPUProfile()

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	for _, mw := range mws {
		mw.Leave()
	}
	mw1.Shutdown()
}

func TestLockMW(nrTimes int, pprofFile io.Writer) {
	mw1, mws := setupHosts(8, 4096, 4096)

	var startTime time.Time
	startTime = time.Now()
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	for i := 0; i < nrTimes; i++ {
		mws[i%7].Lock(0)
		mws[i%7].Release(0)
	}
	pprof.StopCPUProfile()

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	for _, mw := range mws {
		mw.Leave()
	}
	mw1.Shutdown()
}

func TestSynchronizedWritesMW(nrhosts, nrRounds int, pprofFile io.Writer) {
	mw1, mws := setupHosts(nrhosts, 4096, 4096)
	addr, _ := mw1.Malloc(1)

	var startTime time.Time
	startTime = time.Now()
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}

	for i := 0; i < nrRounds; i++ {
		mw1.Write(addr, byte(1))
		readOnAllHosts(addr, mws)
	}
	pprof.StopCPUProfile()

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	for _, mw := range mws {
		mw.Leave()
	}
	mw1.Shutdown()
}

func TestNonSynchronizedReadWritesMW(nrRounds int, pprofFile io.Writer) {
	mw1, _ := setupHosts(1, 4096, 4096)
	addr, _ := mw1.Malloc(1)
	mw1.Write(addr, byte(1))
	var startTime time.Time
	startTime = time.Now()
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}

	for i := 0; i < nrRounds; i++ {
		mw1.Write(addr, byte(1))
		mw1.Read(addr)
	}

	end := time.Now()
	diff := end.Sub(startTime)
	fmt.Println("execution time:", diff.String())
	mw1.Shutdown()
}

func readOnAllHosts(addr int, mws []*multiview.Multiview) {
	for _, mw := range mws {
		mw.Read(addr)
	}
}

func setupHosts(nrHosts int, memSize, pageByteSize int) (manager *multiview.Multiview, mws []*multiview.Multiview) {
	manager = multiview.NewMultiView()
	manager.Initialize(memSize, pageByteSize, nrHosts)
	mws = make([]*multiview.Multiview, nrHosts-1)
	for i := range mws {
		mws[i] = multiview.NewMultiView()
		mws[i].Join(memSize, pageByteSize)
	}
	return
}
