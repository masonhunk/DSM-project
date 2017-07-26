package Benchmarks

import (
	"DSM-project/multiview"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestJacobiProgramMultiView(t *testing.T) {
	//log.SetOutput(ioutil.Discard)
	group := sync.WaitGroup{}
	group.Add(2)
	go JacobiProgramMultiView(4, 2, true, group)
	go func() {
		time.Sleep(time.Millisecond * 200)
		JacobiProgramMultiView(4, 2, false, group)
	}()
	group.Wait()
}

func JacobiProgramMultiView(nrIterations int, nrProcs int, isManager bool, group sync.WaitGroup) {
	testMatrix := [][]int{
		{3, 2, 4, 7},
		{2, 3, 6, 4},
		{4, 6, 7, 5},
		{7, 4, 5, 2},
	}
	const M = 4
	const N = 4
	const float32_BYTE_LENGTH = 4 //32 bits
	var privateArray [][]float32  //privateArray[M][N]
	privateArray = make([][]float32, M)
	for i := range privateArray {
		privateArray[i] = make([]float32, N)
	}

	mw := multiview.NewMultiView()
	if isManager {
		mw.Initialize(N*M*float32_BYTE_LENGTH, 4, nrProcs)
	} else {
		mw.Join(M*N*float32_BYTE_LENGTH, 4)
	}
	mw.Barrier(0)

	length := M / nrProcs
	begin := length * int(mw.Id-1)
	end := length * int(mw.Id)
	fmt.Println("begin, end:", begin, end)

	//allocate the float entries as minipages
	gridEntryAddresses := make([][]int, end+1)
	for i := range gridEntryAddresses {
		gridEntryAddresses[i] = make([]int, N)
	}
	for i := begin; i < end; i++ {
		row := gridEntryAddresses[i]
		for j := range row {
			gridEntryAddresses[i][j], _ = mw.Malloc(float32_BYTE_LENGTH)
			//placeholder value
			var valAsBytes []byte = float32ToBytes(float32(testMatrix[i][j]))
			for r, b := range valAsBytes {
				mw.Write(gridEntryAddresses[i][j]+r, b)
			}
		}
	}

	mw.Barrier(1)

	for iter := 1; iter <= nrIterations; iter++ {
		fmt.Println("in iteration nr", iter, "at process", mw.Id)
		for i := begin; i < end; i++ {
			for j := 0; j < N; j++ {
				divisionAmount := 4
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
					g2, _ = mw.ReadBytes(gridEntryAddresses[i+1][j], float32_BYTE_LENGTH)
				} else {
					divisionAmount--
				}
				if j > 0 {
					g3, _ = mw.ReadBytes(gridEntryAddresses[i][j-1], float32_BYTE_LENGTH)
				}
				if j < N-1 {
					g4, _ = mw.ReadBytes(gridEntryAddresses[i][j+1], float32_BYTE_LENGTH)
				} else {
					divisionAmount--
				}
				privateArray[i][j] = (bytesToFloat32(g1) + bytesToFloat32(g2) + bytesToFloat32(g3) + bytesToFloat32(g4)) / 4
			}
			for i := begin; i < end; i++ {
				for j := 0; j < N; j++ {
					addr := gridEntryAddresses[i][j]
					var valAsBytes []byte = float32ToBytes(privateArray[i][j])
					for r, b := range valAsBytes {
						mw.Write(addr+r, b)
					}
				}
			}
		}
	}
	defer func() {
		if isManager {
			mw.Lock(0)
			fmt.Println("result at host 1")
			for i := begin; i <= end; i++ {
				row := gridEntryAddresses[i]
				for j := range row {
					res, _ := mw.ReadBytes(gridEntryAddresses[i][j], float32_BYTE_LENGTH)
					fmt.Print(bytesToFloat32(res), ",")
				}
				fmt.Println()
			}
			mw.Release(0)
			mw.Barrier(2)
			mw.Leave()
			mw.Shutdown()

		} else {
			mw.Lock(0)
			fmt.Println("result at host 2")
			for i := begin; i < end; i++ {
				row := gridEntryAddresses[i]
				for j := range row {
					res, _ := mw.ReadBytes(gridEntryAddresses[i][j], float32_BYTE_LENGTH)
					fmt.Print(bytesToFloat32(res), ",")
				}
				fmt.Println()
			}
			mw.Release(0)
			mw.Barrier(2)
			mw.Leave()
		}
		fmt.Println("exiting algorithm...")
		group.Done()
		return
	}()

}
