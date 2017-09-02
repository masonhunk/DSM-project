package Benchmarks

import (
	"DSM-project/multiview"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
)

var _ = log.Print
var _ = log.Print

func TestJacobiProgramMultiView(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	group := sync.WaitGroup{}
	pageSize := 4096
	//decent results with N,M = 64, iterations = 10-24
	nrIterations := 20
	nrProcs := 4
	runtime.GOMAXPROCS(nrProcs) // or 2 or 4
	group.Add(nrProcs)
	matrixsize := 64
	go JacobiProgramMultiView(matrixsize, nrIterations, nrProcs, true, pageSize, &group, nil)
	for i := 0; i < nrProcs-1; i++ {
		go func() {
			time.Sleep(190 * time.Millisecond)
			go JacobiProgramMultiView(matrixsize, nrIterations, nrProcs, false, pageSize, &group, nil)
		}()
	}
	group.Wait()
}

func JacobiProgramMultiView(matrixSize int, nrIterations int, nrProcs int, isManager bool, pageByteSize int, group *sync.WaitGroup, pprofFile io.Writer) {
	/*testMatrix := [][]int{
		{5, 6, 6, 2, 5, 6, 9, 2},
		{6, 5, 9, 5, 5, 6, 3, 7},
		{6, 9, 7, 8, 4, 4, 6, 2},
		{2, 5, 8, 7, 7, 7, 8, 3},
		{5, 5, 4, 7, 9, 5, 3, 2},
		{6, 6, 4, 7, 5, 8, 3, 5},
		{9, 3, 6, 8, 3, 3, 4, 5},
		{2, 7, 2, 3, 2, 5, 5, 7},
	}*/
	var M = matrixSize
	var N = matrixSize
	const float32_BYTE_LENGTH = 4 //32 bits
	var privateArray [][]float32  //privateArray[M][N]
	privateArray = make([][]float32, M)
	for i := range privateArray {
		privateArray[i] = make([]float32, N)
	}

	//allocate the float entries as minipages
	gridEntryAddresses := make([][]int, M)
	for i := range gridEntryAddresses {
		gridEntryAddresses[i] = make([]int, N)
	}
	mw := multiview.NewMultiView()
	if isManager {
		var setupStart time.Time = time.Now()

		mw.Initialize(N*M*float32_BYTE_LENGTH, pageByteSize, nrProcs)
		mw.CSVLoggingIsEnabled(false)

		allocs := make([]int, M*N)
		for i := range allocs {
			allocs[i] = float32_BYTE_LENGTH
		}
		addrs, _ := mw.MultiMalloc(allocs)
		for i := range gridEntryAddresses {
			row := gridEntryAddresses[i]
			for j := range row {
				gridEntryAddresses[i][j] = addrs[i+j]
				if gridEntryAddresses[i][j] < 0 {
					panic(fmt.Sprintln("Address was negative: ", i, ", ", j, ", ", gridEntryAddresses[i][j]))
				}
				//gridEntryAddresses[i][j], _ = mw.Malloc(float32_BYTE_LENGTH)
				//placeholder value
				/*var valAsBytes []byte = float32ToBytes(20.0)
				for r, b := range valAsBytes {
					mw.Write(gridEntryAddresses[i][j]+r, b)
				}*/
			}
		}
		fmt.Println("Manager done with setup after", time.Now().Sub(setupStart))
		mw.Barrier(0)
	} else {
		mw.Join(M*N*float32_BYTE_LENGTH, pageByteSize)
		//calculate the addresses of the pointers allocated by the manager host
		mw.Barrier(0)
		for i := range gridEntryAddresses {
			row := gridEntryAddresses[i]
			for j := range row {
				k := i * N * float32_BYTE_LENGTH
				l := k + j*float32_BYTE_LENGTH
				q := (i + j) % (mw.GetPageSize() / float32_BYTE_LENGTH)
				memsize := M * N * float32_BYTE_LENGTH
				vAddr := (q+1)*memsize + l
				gridEntryAddresses[i][j] = vAddr
			}
		}
	}
	var startTime time.Time
	if mw.Id == 1 {
		startTime = time.Now()
	}
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	if isManager {
		mw.CSVLoggingIsEnabled(true)
	}
	length := M / nrProcs
	begin := length * int(mw.Id-1)
	end := length * int(mw.Id)
	if end <= begin {
		panic("begin is larger than end")
	}
	fmt.Println("begin, end at host", mw.Id, ":", begin, end)
	mw.SetShouldLogNetwork(true)
	//fmt.Println("at barrier 1")
	mw.Barrier(1)

	for iter := 1; iter <= nrIterations; iter++ {
		fmt.Println("in iteration", iter, "at host", mw.Id)
		iterationStartTime := time.Now()
		for i := begin; i < end; i++ {
			for j := 0; j < N; j++ {
				var divisionAmount int = 4
				g1 := []byte{0, 0, 0, 0}
				g2 := []byte{0, 0, 0, 0}
				g3 := []byte{0, 0, 0, 0}
				g4 := []byte{0, 0, 0, 0}

				if i > 0 {
					g1, _ = mw.ReadBytes(gridEntryAddresses[i-1][j], float32_BYTE_LENGTH)
				} else {
					divisionAmount--
				}
				if i < M-1 {
					if i+1 == end {
						//log.Println("about to read to shared variable with i+1, j values:", i+1, j, "and address", gridEntryAddresses[i+1][j])
					}
					g2, _ = mw.ReadBytes(gridEntryAddresses[i+1][j], float32_BYTE_LENGTH)
				} else {
					divisionAmount--
				}
				if j > 0 {
					g3, _ = mw.ReadBytes(gridEntryAddresses[i][j-1], float32_BYTE_LENGTH)
				} else {
					divisionAmount--
				}
				if j < N-1 {
					g4, _ = mw.ReadBytes(gridEntryAddresses[i][j+1], float32_BYTE_LENGTH)
				} else {
					divisionAmount--
				}
				privateArray[i][j] = (bytesToFloat32(g1) + bytesToFloat32(g2) + bytesToFloat32(g3) + bytesToFloat32(g4)) / float32(divisionAmount)
			}
		}
		mw.Barrier(2)
		//fmt.Println("after barrier 2 at host", mw.Id)
		for i := begin; i < end; i++ {
			for j := 0; j < N; j++ {
				addr := gridEntryAddresses[i][j]
				var valAsBytes []byte = float32ToBytes(privateArray[i][j])
				mw.WriteBytes(addr, valAsBytes)
			}
		}
		mw.Barrier(3)
		fmt.Println("iteration", iter, " took", time.Now().Sub(iterationStartTime))
		log.Println("done with iteration", iter)
	}
	mw.Barrier(4)
	log.Println("exiting algorithm at process", mw.Id, "...")
	defer func() {
		resultMatrix := make([][]float32, M)
		for i := range resultMatrix {
			resultMatrix[i] = make([]float32, N)
		}

		if isManager {
			mw.CSVLoggingIsEnabled(false)
			end := time.Now()
			diff := end.Sub(startTime)
			fmt.Println("execution time:", diff.String())

			mw.Lock(0)
			//fmt.Println("result at manager:")
			for i := range resultMatrix {
				row := resultMatrix[i]
				for j := range row {
					res, _ := mw.ReadBytes(gridEntryAddresses[i][j], float32_BYTE_LENGTH)
					resultMatrix[i][j] = bytesToFloat32(res)
				}
				//fmt.Println(row)
			}
			//fmt.Println("result at host", mw.Id, ":", resultMatrix)
			mw.Release(0)
			mw.Barrier(5)
			mw.Shutdown()
			log.Println("arrived at shutdown")

		} else {
			//fmt.Println("result at host", mw.Id)
			mw.Lock(0)
			/*for i := range resultMatrix {
				row := resultMatrix[i]
				for j := range row {
					res, _ := mw.ReadBytes(gridEntryAddresses[i][j], float32_BYTE_LENGTH)
					resultMatrix[i][j] = bytesToFloat32(res)
				}
				//fmt.Println(row)
			}*/

			mw.Release(0)
			mw.Barrier(5)
			mw.Leave()
		}
		group.Done()
		return
	}()

}
