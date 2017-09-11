package Benchmarks

import (
	"DSM-project/dsm-api/treadmarks"
	"DSM-project/multiview"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
)

func TestParallelSumTM(t *testing.T) {
	runtime.GOMAXPROCS(4) // or 2 or 4
	log.SetOutput(ioutil.Discard)
	group := sync.WaitGroup{}
	pageSize := 4096
	var nrOfInts int64 = 4096 * 1000000
	var batchSize int64 = 10000 * 4096 // nr of ints in batch
	nrProcs := 4
	group.Add(nrProcs)
	go ParallelSumTM(batchSize, nrOfInts, nrProcs, true, 2000, pageSize, &group, nil)
	for i := 0; i < nrProcs-1; i++ {
		go func(i int) {
			time.Sleep(150 * time.Millisecond)
			ParallelSumTM(batchSize, nrOfInts, nrProcs, false, 2000+i+10, pageSize, &group, nil)
		}(i)
	}
	group.Wait()

}

func TestParallelSumMW(t *testing.T) {
	runtime.GOMAXPROCS(4) // or 2 or 4
	log.SetOutput(ioutil.Discard)
	group := sync.WaitGroup{}
	pageSize := 4096
	nrOfInts := 4096 * 1000000
	batchSize := 10000 * 4096 // nr of ints in batch
	nrProcs := 4
	group.Add(nrProcs)
	go ParallelSumMW(batchSize, nrOfInts, nrProcs, true, pageSize, &group, nil)
	for i := 0; i < nrProcs-1; i++ {
		go func() {
			time.Sleep(150 * time.Millisecond)
			ParallelSumMW(batchSize, nrOfInts, nrProcs, false, pageSize, &group, nil)
		}()
	}
	group.Wait()

}

func ParallelSumMW(batchSize int, nrOfInts int, nrProcs int, isManager bool, pageByteSize int, group *sync.WaitGroup, cpuProfFile io.Writer) {
	const INT_BYTE_LENGTH = 8 //64 bits
	var sharedSumAddr int
	var currBatchNrAddr int
	mw := multiview.NewMultiView()
	if isManager {
		mw.Initialize(INT_BYTE_LENGTH*(2), pageByteSize, nrProcs)
		log.Println("started manager host with memory byte size", mw.GetMemoryByteSize())
		mw.Barrier(0)
		rand.Seed(time.Now().UnixNano())
		sharedSumAddr, _ = mw.Malloc(INT_BYTE_LENGTH)
		currBatchNrAddr, _ = mw.Malloc(INT_BYTE_LENGTH)
		mw.WriteInt64(sharedSumAddr, 0)
		mw.WriteInt64(currBatchNrAddr, 0)
		log.Println("sharedSumAddr at manager:", sharedSumAddr)
		log.Println("currBatchNrAddr at manager:", currBatchNrAddr)
		mw.Barrier(3)

	} else {
		mw.Join(INT_BYTE_LENGTH*(2), pageByteSize)
		mw.Barrier(0)
		mw.Barrier(3)

		sharedSumAddr = mw.GetMemoryByteSize()
		currBatchNrAddr = 2*mw.GetMemoryByteSize() + 8
		log.Println("sharedSumAddr at host", mw.Id, sharedSumAddr)
		log.Println("currBatchNrAddr at host:", mw.Id, currBatchNrAddr)
	}
	var startTime time.Time
	if mw.Id == 1 {
		startTime = time.Now()
	}

	if cpuProfFile != nil {
		if err := pprof.StartCPUProfile(cpuProfFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	mw.Barrier(1) //ensures everyone gets their first lock

	localSum := 1
	for {
		//lock and get next batch and calculate localSum
		mw.Lock(1)
		currBatchNr := mw.ReadInt64(currBatchNrAddr)
		if currBatchNr >= nrOfInts {
			mw.Release(1)
			break
		}
		mw.WriteInt64(currBatchNrAddr, currBatchNr+batchSize)
		mw.Release(1)
		for i := 0; i < batchSize; i++ {
			localSum = ((localSum * localSum) % 2000000000) +33
		}

	}
	mw.Lock(2)
	localSum = ((localSum * mw.ReadInt64(sharedSumAddr)) % 2000000000) +33
	mw.WriteInt64(sharedSumAddr, localSum)
	mw.Release(2)
	mw.Barrier(2)
	if mw.Id == 1 {
		end := time.Now()
		diff := end.Sub(startTime)
		fmt.Println("execution time:", diff.String())
		fmt.Println("result localSum:", mw.ReadInt64(sharedSumAddr))
	}
	mw.Barrier(3)
	log.Println("exiting algorithm at process", mw.Id, "...")
	defer func() {
		if isManager {
			mw.Shutdown()
		} else {
			mw.Leave()
		}
		group.Done()
	}()
}

func ParallelSumTM(batchSize int64, nrOfInts int64, nrProcs int, isManager bool, port int, pageByteSize int, group *sync.WaitGroup, cpuProfFile io.Writer) {
	const INT_BYTE_LENGTH = 8 //64 bits
	var sharedSumAddr int
	var currBatchNrAddr int
	var tm *treadmarks.TreadmarksApi
	tm, _ = treadmarks.NewTreadmarksApi(INT_BYTE_LENGTH*2, pageByteSize, uint8(nrProcs), uint8(4), uint8(4))
	tm.Initialize(port)
	defer tm.Shutdown()
	if isManager {
		tm.Barrier(0)
		rand.Seed(time.Now().UnixNano())
		sharedSumAddr, _ = tm.Malloc(INT_BYTE_LENGTH)
		currBatchNrAddr, _ = tm.Malloc(INT_BYTE_LENGTH)
		fmt.Println("sharedSumAddr at manager:", sharedSumAddr)
		fmt.Println("currBatchNrAddr at manager:", currBatchNrAddr)
		tm.Barrier(3)

	} else {
		tm.Join("localhost", 2000)
		tm.Barrier(0)
		tm.Barrier(3)

		sharedSumAddr = 0
		currBatchNrAddr = 8
		fmt.Println("sharedSumAddr at host", tm.GetId(), sharedSumAddr)
		fmt.Println("currBatchNrAddr at host", tm.GetId(), currBatchNrAddr)
	}
	var startTime time.Time
	if isManager {
		startTime = time.Now()
	}

	if cpuProfFile != nil {
		if err := pprof.StartCPUProfile(cpuProfFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	tm.SetLogging(true)
	tm.Barrier(1) //ensures everyone gets their first lock

	var prevBatchNumber int64
	var localSum int64 = 1
	for {
		//lock and get next batch and calculate localSum
		tm.AcquireLock(1)
		currBatchNr := readInt64(tm, currBatchNrAddr)
		if prevBatchNumber > currBatchNr {
			fmt.Println(currBatchNr)
		}
		prevBatchNumber = currBatchNr

		if currBatchNr >= nrOfInts {
			tm.ReleaseLock(1)
			break
		}
		//tm.WriteInt64(currBatchNrAddr, currBatchNr+batchSize)
		writeInt64(tm, currBatchNrAddr, currBatchNr+batchSize)
		tm.ReleaseLock(1)
		var i int64
		for i = 0; i < batchSize; i++ {
			localSum = ((localSum * localSum) % 2000000000)+33
		}

	}
	tm.AcquireLock(2)
	localSum = ((localSum * readInt64(tm, sharedSumAddr)) % 2000000000) +33
	writeInt64(tm, sharedSumAddr, int64(localSum))
	tm.ReleaseLock(2)
	tm.Barrier(2)
	if isManager {
		end := time.Now()
		diff := end.Sub(startTime)
		fmt.Println("execution time:", diff.String())
		fmt.Println("result localSum:", readInt64(tm, sharedSumAddr))
	}
	tm.Barrier(3)
	log.Println("exiting algorithm at process", tm.GetId(), "...")
	defer group.Done()
}
