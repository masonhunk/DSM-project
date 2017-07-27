package Benchmarks

import (
	"DSM-project/multiview"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestMergeSortMW(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	group := sync.WaitGroup{}
	group.Add(4)
	start := time.Now()
	arraySize := 4096 * 10000
	go MergeSortMW(arraySize, 4, true, 4096, &group)
	go func() {
		time.Sleep(time.Millisecond * 200)
		MergeSortMW(arraySize, 4, false, 4096, &group)
	}()
	go func() {
		time.Sleep(time.Millisecond * 200)
		MergeSortMW(arraySize, 4, false, 4096, &group)
	}()
	go func() {
		time.Sleep(time.Millisecond * 200)
		MergeSortMW(arraySize, 4, false, 4096, &group)
	}()
	group.Wait()
	end := time.Now()
	diff := end.Sub(start)
	fmt.Println("execution time:", diff.String())
}

func MergeSortMW(arraySize int, nrProcs int, isManager bool, pageByteSize int, group *sync.WaitGroup) {
	var ARRAY_SIZE = arraySize
	const INT_BYTE_LENGTH = 4 //32 bits
	var privateArray []int
	privateArray = make([]int, ARRAY_SIZE/nrProcs)
	fmt.Println("private array size:", len(privateArray))
	cellByteSize := ARRAY_SIZE * INT_BYTE_LENGTH / nrProcs
	arraySectionAddresses := make([]int, ARRAY_SIZE*INT_BYTE_LENGTH/cellByteSize)
	fmt.Println("cell byte size:", cellByteSize)
	fmt.Println("arraySectionAddresses length:", len(arraySectionAddresses))

	mw := multiview.NewMultiView()
	if isManager {
		mw.Initialize(INT_BYTE_LENGTH*ARRAY_SIZE, pageByteSize, nrProcs)
		rand.Seed(time.Now().UnixNano())
		for i := range arraySectionAddresses {
			arraySectionAddresses[i], _ = mw.Malloc(cellByteSize)
			fmt.Println("got addr from malloc:", arraySectionAddresses[i])
			//fill with random value
			for j := 0; j < cellByteSize/INT_BYTE_LENGTH; j++ {
				rn := rand.Intn(1000000)
				mw.WriteInt(arraySectionAddresses[i]+j*INT_BYTE_LENGTH, rn)
			}
		}
		fmt.Println("address array at manager:", arraySectionAddresses)
		mw.Barrier(0)

	} else {
		mw.Join(ARRAY_SIZE*INT_BYTE_LENGTH, pageByteSize)
		mw.Barrier(0)
		//calculate the addresses of the pointers allocated by the manager host
		for i, _ := range arraySectionAddresses {
			k := cellByteSize * i
			q := 0
			if cellByteSize%pageByteSize != 0 {
				q = i % max((pageByteSize/cellByteSize), 2)
			}
			vAddr := (q+1)*mw.GetMemoryByteSize() + k
			log.Println("k,q, vAddr:", k, q, vAddr)
			arraySectionAddresses[i] = vAddr
			mw.ReadInt(vAddr)
		}
	}
	mw.Barrier(1)
	start := cellByteSize * (int(mw.Id) - 1)
	startingAddr := arraySectionAddresses[int(mw.Id)-1]

	mw.Lock(int(mw.Id))
	mw.Barrier(2) //ensures everyone gets their first lock
	//sort here
	for i := range privateArray {
		privateArray[i] = mw.ReadInt(startingAddr + i*INT_BYTE_LENGTH)
	}
	var sortableArray sort.IntSlice = privateArray
	sort.Sort(sortableArray)
	mergeArrSize := cellByteSize * 2
	level := 1
	for mergeArrSize <= ARRAY_SIZE {
		if start%mergeArrSize == 0 { //if I am the leftmost process in the subtree, I win and do the sorting
			otherProcId := int(mw.Id) + level
			mw.Lock(otherProcId)
			//read losing processor's result into private array and sort locally
			otherProcRes := make([]int, mergeArrSize/2)
			for i := 0; i < mergeArrSize/2; i++ {
				otherProcRes[i] = mw.ReadInt(arraySectionAddresses[otherProcId] + i*INT_BYTE_LENGTH)
			}
			privateArray = mergeArrays(privateArray, otherProcRes)
			level = level * 2
			mergeArrSize = mergeArrSize * 2
		} else {
			//only write results to DSM once the lock is about to be released
			for i, val := range privateArray {
				mw.WriteInt(startingAddr+i*INT_BYTE_LENGTH, val)
			}
			break
		}
	}
	mw.Release(int(mw.Id))
	mw.Barrier(3)
	log.Println("exiting algorithm at process", mw.Id, "...")
	defer func() {
		if isManager {
			//fmt.Println("result array:",privateArray)
			//var res sort.IntSlice = privateArray
			//fmt.Println("isSorted:", sort.IsSorted(res))
			mw.Shutdown()
		} else {
			mw.Leave()
		}
		group.Done()
	}()
}

func TestMergArrays(t *testing.T) {
	a := []int{1, 2, 3, 4, 5, 6, 7, 8}
	b := []int{4, 6, 8, 9, 10}
	res := mergeArrays(a, b)
	fmt.Println(res)
}

func TestRandomness(t *testing.T) {
	arrayAddresses := make([][]byte, 6)
	rand.Seed(time.Now().UnixNano())
	for i := range arrayAddresses {
		bytes := make([]byte, 4)
		rand.Read(bytes)
		arrayAddresses[i] = bytes
	}
	fmt.Println(arrayAddresses)
}

func TestBubblesort(t *testing.T) {
	var toBeSorted sort.IntSlice = sort.IntSlice{9, 3, 2, 4, 8, 6, 7, 22, 0}
	bubbleSort(toBeSorted)
	fmt.Print(toBeSorted)
	assert.True(t, sort.IsSorted(toBeSorted))
}

func bubbleSort(input []int) {
	// n is the number of items in our list
	n := len(input)
	swapped := true
	for swapped {
		swapped = false
		for i := 1; i < n; i++ {
			if input[i-1] > input[i] {
				// swap values using Go's tuple assignment
				input[i], input[i-1] = input[i-1], input[i]
				swapped = true
			}
		}
	}
}
