package Benchmarks

import (
	"DSM-project/memory"
	"DSM-project/treadmarks"
	"encoding/binary"
	"fmt"
	"math/rand"
)

func setupTreadMarksStruct1(nrProcs, memsize, pagebytesize, nrlocks, nrbarriers int) *treadmarks.TreadMarks {
	vm1 := memory.NewVmem(memsize, pagebytesize)
	tm1 := treadmarks.NewTreadMarks(vm1, nrProcs, nrlocks, nrbarriers)
	return tm1
}

func SortedIntBenchmark(nrProcs int, isManager bool, N int, Bmax uint32, Imax int) {
	//First we do setup.
	rand.Seed(314159265)
	tm := setupTreadMarksStruct1(nrProcs, (((N*4)/4096)+1)*4096, 4096, 2, 2)
	key := func(i int) int { return i * 4 }
	if isManager {
		tm.Startup()
		for i := 0; i < N; i++ {
			ki := uint64(float64(Bmax) * ((rand.Float64() + rand.Float64() + rand.Float64() + rand.Float64()) / 4))
			tm.WriteBytes(key(i), intToBytes(ki))
		}
	} else {
		tm.Join("localhost:2000")
		fmt.Println("joined with id: ", tm.ProcId)
	}

	defer tm.Shutdown()
}

func intToBytes(i uint64) []byte {
	result := make([]byte, 8)
	binary.PutUvarint(result, i)
	return result
}
