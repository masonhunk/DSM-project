package Benchmarks

import (
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

func TestParallelSum(t *testing.T) {
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
			localSum = (localSum * localSum) % 2000000000
		}

	}
	mw.Lock(2)
	localSum = (localSum * mw.ReadInt64(sharedSumAddr)) % 2000000000
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
