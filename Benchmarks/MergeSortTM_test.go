package Benchmarks

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestMergeSortTM(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	group := sync.WaitGroup{}
	start := time.Now()
	arraySize := 10
	nrProcs := 2
	group.Add(nrProcs)
	go MergeSortTM(arraySize, nrProcs, true, 4096, &group)
	for i := 0; i < nrProcs-1; i++ {
		go func() {
			time.Sleep(time.Millisecond * 200)
			MergeSortTM(arraySize, nrProcs, false, 4096, &group)
		}()
	}
	group.Wait()
	end := time.Now()
	diff := end.Sub(start)
	fmt.Println("execution time:", diff.String())
}

func MergeSortTM(arraySize int, nrProcs int, isManager bool, pageByteSize int, group *sync.WaitGroup) {
	var ARRAY_SIZE = arraySize
	const INT_BYTE_LENGTH = 4 //32 bits
	var privateArray []int
	privateArray = make([]int, ARRAY_SIZE/nrProcs)
	fmt.Println("private array size:", len(privateArray))
	cellByteSize := ARRAY_SIZE * INT_BYTE_LENGTH / nrProcs
	fmt.Println("cell byte size:", cellByteSize)

	tm := setupTreadMarksStruct(nrProcs, ARRAY_SIZE*INT_BYTE_LENGTH, pageByteSize, nrProcs, nrProcs+1)
	if isManager {
		tm.Startup()
		rand.Seed(time.Now().UnixNano())
		for i := 0; i < ARRAY_SIZE*INT_BYTE_LENGTH/cellByteSize; i++ {
			addr, _ := tm.Malloc(cellByteSize)
			log.Println("got addr from malloc:", addr)
			//fill with random values
			for j := 0; j < cellByteSize/INT_BYTE_LENGTH; j++ {
				rn := rand.Intn(1000000)
				tm.WriteInt(addr+j*INT_BYTE_LENGTH, rn)
			}
		}
		tm.Barrier(0)

	} else {
		tm.Join("localhost:2000")
		tm.Barrier(0)
		start := cellByteSize * (int(tm.ProcId) - 1)
		//calculate the addresses of the pointers allocated by the manager host
		for i := 0; i < cellByteSize; i++ {
			fmt.Println("reading at address", start+(i*INT_BYTE_LENGTH))
			tm.ReadInt(start + (i * INT_BYTE_LENGTH))
		}
	}
	start := cellByteSize * (int(tm.ProcId) - 1)
	tm.AcquireLock(int(tm.ProcId))
	tm.Barrier(1) //ensures everyone gets their first lock
	//sort own array here
	for i := range privateArray {
		privateArray[i] = tm.ReadInt(start + i*INT_BYTE_LENGTH)
	}
	var sortableArray sort.IntSlice = privateArray
	sort.Sort(sortableArray)
	mergeArrSize := cellByteSize * 2
	level := 1
	for mergeArrSize <= ARRAY_SIZE {
		if start%mergeArrSize == 0 { //if I am the leftmost process in the subtree, I win and do the sorting
			otherProcId := int(tm.ProcId) + level
			tm.AcquireLock(otherProcId)
			//read losing processor's result into private array and merge them
			otherProcRes := make([]int, mergeArrSize/2)
			otherProcAddr := cellByteSize * (int(otherProcId) - 1)
			for i := 0; i < mergeArrSize/2; i++ {
				otherProcRes[i] = tm.ReadInt(otherProcAddr + i*INT_BYTE_LENGTH)
			}
			privateArray = mergeArrays(privateArray, otherProcRes)
			level = level * 2
			mergeArrSize = mergeArrSize * 2
		} else {
			//only write results to DSM once the lock is about to be released
			for i, val := range privateArray {
				tm.WriteInt(start+i*INT_BYTE_LENGTH, val)
			}
			break
		}
	}
	tm.ReleaseLock(int(tm.ProcId))
	tm.Barrier(2)
	defer func() {
		if isManager {
			//fmt.Println("result array:",privateArray)
			//var res sort.IntSlice = privateArray
			//fmt.Println("isSorted:", sort.IsSorted(res))
			tm.Shutdown()
		} else {
			tm.Close()
		}
		group.Done()
	}()
}
