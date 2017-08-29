package Benchmarks

import (
	"DSM-project/dsm-api/treadmarks"
	"fmt"
	"io"
	"log"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
)

func TestJacobiProgramTreadMarks(t *testing.T) {
	//log.SetOutput(ioutil.Discard)
	group := new(sync.WaitGroup)
	group.Add(2)
	matrixsize := 64
	go JacobiProgramTreadMarks(matrixsize, 4, 2, true, 2000, group, nil)
	go func() {
		time.Sleep(time.Millisecond * 200)
		JacobiProgramTreadMarks(matrixsize, 4, 2, false, 2001, group, nil)
	}()
	group.Wait()
}

func JacobiProgramTreadMarks(matrixsize int, nrIterations int, nrProcs int, isManager bool, port int, group *sync.WaitGroup, pprofFile io.Writer) {
	var M = matrixsize
	var N = matrixsize
	const float32_BYTE_LENGTH = 4 //32 bits
	var privateArray [][]float32  //privateArray[M][N]
	privateArray = make([][]float32, M)
	for i := range privateArray {
		privateArray[i] = make([]float32, N)
	}
	gridAddr := func(m, n int) int {
		if m >= M || n >= N || m < 0 || n < 0 {
			return -1
		}
		return (m * N * float32_BYTE_LENGTH) + (n * float32_BYTE_LENGTH)
	}

	tm, _ := treadmarks.NewTreadmarksApi(M*N*float32_BYTE_LENGTH, 4096, uint8(nrProcs), uint8(nrProcs), uint8(nrProcs))
	tm.Initialize(port)
	if !isManager {
		tm.Join("localhost", 2000)
		fmt.Println("joined with id:", tm.GetId())
	}

	tm.Barrier(0)
	var startTime time.Time
	if isManager {
		startTime = time.Now()
	}
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	length := M / nrProcs
	begin := length * int(tm.GetId())
	end := length * int(tm.GetId()+1)
	fmt.Println("begin, end at host", tm.GetId(), ":", begin, end)
	if end <= begin {
		panic("begin is larger than end")
	}
	for iter := 1; iter <= nrIterations; iter++ {
		fmt.Println("in iteration nr", iter)
		for i := begin; i < end; i++ {
			for j := 0; j < N; j++ {
				divisionAmount := 4
				g1 := []byte{0, 0, 0, 0}
				g2 := []byte{0, 0, 0, 0}
				g3 := []byte{0, 0, 0, 0}
				g4 := []byte{0, 0, 0, 0}

				if i > 0 {
					g1 = readBytes(tm, gridAddr(i-1, j), float32_BYTE_LENGTH)

				} else {
					divisionAmount--
				}
				if i < M-1 {
					if i+1 == end {
						//log.Println("about to read to shared variable with i+1, j values:", i+1, j, "and address", gridEntryAddresses[i+1][j])
					}
					g2 = readBytes(tm, gridAddr(i+1, j), float32_BYTE_LENGTH)
				} else {
					divisionAmount--
				}
				if j > 0 {
					g3 = readBytes(tm, gridAddr(i, j-1), float32_BYTE_LENGTH)
				} else {
					divisionAmount--
				}
				if j < N-1 {
					g4 = readBytes(tm, gridAddr(i, j+1), float32_BYTE_LENGTH)
				} else {
					divisionAmount--
				}
				privateArray[i][j] = (bytesToFloat32(g1) + bytesToFloat32(g2) + bytesToFloat32(g3) + bytesToFloat32(g4)) / float32(divisionAmount)
			}
		}
		//fmt.Println("at barrier 1 in iteration", iter)
		tm.Barrier(1)
		for i := begin; i < end; i++ {
			for j := 0; j < N; j++ {
				addr := gridAddr(i, j)
				var valAsBytes []byte = float32ToBytes(privateArray[i][j])
				if len(valAsBytes) > 8 {
					panic("valAsBytes was longer than expected")
				}
				for r, b := range valAsBytes {
					tm.Write(addr+r, b)
				}
			}
		}
		//fmt.Println("at barrier 2 in iteration", iter)
		tm.Barrier(2)
		//fmt.Println("after barrier 2 in iteration", iter)
	}
	if isManager {
		endTime := time.Now()
		diff := endTime.Sub(startTime)
		fmt.Println("execution time:", diff.String())
	}

	fmt.Println("before done")
	tm.Shutdown()
	group.Done()
	fmt.Println("after done")
}
