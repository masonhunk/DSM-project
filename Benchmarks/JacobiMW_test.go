package Benchmarks

import (
	"DSM-project/multiview"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
)

var _ = log.Print
var _ = log.Print

func TestJacobiProgramMultiView(t *testing.T) {
	//log.SetOutput(ioutil.Discard)
	group := sync.WaitGroup{}
	start := time.Now()
	pageSize := 4096
	nrIterations := 5
	nrProcs := 8
	group.Add(nrProcs)
	group.Add(nrProcs)
	go JacobiProgramMultiView(nrIterations, nrProcs, true, pageSize, &group)
	for i := 0; i < nrProcs-1; i++ {
		go func() {
			time.Sleep(150 * time.Millisecond)
			go JacobiProgramMultiView(nrIterations, nrProcs, false, pageSize, &group)
		}()
	}
	group.Wait()
	end := time.Now()
	diff := end.Sub(start)
	fmt.Println("execution time:", diff.String())
}

func JacobiProgramMultiView(nrIterations int, nrProcs int, isManager bool, pageByteSize int, group *sync.WaitGroup) {
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
	const M = 16
	const N = 16
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
		mw.Initialize(N*M*float32_BYTE_LENGTH, pageByteSize, nrProcs)

		for i := range gridEntryAddresses {
			row := gridEntryAddresses[i]
			for j := range row {
				gridEntryAddresses[i][j], _ = mw.Malloc(float32_BYTE_LENGTH)
				//placeholder value
				var valAsBytes []byte = float32ToBytes(20.0)
				for r, b := range valAsBytes {
					mw.Write(gridEntryAddresses[i][j]+r, b)
				}
			}
		}
		log.Println("manager done writing to grid entry with result:", gridEntryAddresses)
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
				mw.Read(vAddr)
			}
		}
	}

	length := M / nrProcs
	begin := length * int(mw.Id-1)
	end := length * int(mw.Id)
	if end <= begin {
		panic("begin is larger than end")
	}
	mw.Barrier(1)

	for iter := 1; iter <= nrIterations; iter++ {
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
						log.Println("about to read to shared variable with i+1, j values:", i+1, j, "and address", gridEntryAddresses[i+1][j])
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
			mw.Barrier(2)
			for i := begin; i < end; i++ {
				for j := 0; j < N; j++ {
					addr := gridEntryAddresses[i][j]
					var valAsBytes []byte = float32ToBytes(privateArray[i][j])
					mw.WriteBytes(addr, valAsBytes)
				}
			}
			mw.Barrier(3)

		}
	}
	mw.Barrier(4)
	log.Println("exiting algorithm at process", mw.Id, "...")
	defer func() {
		resultMatrix := make([][]float32, M)
		for i := range resultMatrix {
			resultMatrix[i] = make([]float32, N)
		}

		if isManager {
			mw.Lock(0)
			for i := range resultMatrix {
				row := resultMatrix[i]
				for j := range row {
					res, _ := mw.ReadBytes(gridEntryAddresses[i][j], float32_BYTE_LENGTH)
					resultMatrix[i][j] = bytesToFloat32(res)
				}
			}
			log.Println("result at host", mw.Id, ":", resultMatrix)
			mw.Release(0)
			mw.Barrier(5)
			mw.Shutdown()
			log.Println("arrived at shutdown")

		} else {
			mw.Lock(0)
			for i := range resultMatrix {
				row := resultMatrix[i]
				for j := range row {
					res, _ := mw.ReadBytes(gridEntryAddresses[i][j], float32_BYTE_LENGTH)
					resultMatrix[i][j] = bytesToFloat32(res)
				}
			}
			log.Println("result at host", mw.Id, ":", resultMatrix)

			mw.Release(0)
			mw.Barrier(5)
			mw.Leave()
		}
		group.Done()
		return
	}()

}
