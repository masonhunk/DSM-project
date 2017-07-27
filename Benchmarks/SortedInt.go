package Benchmarks

import (
	"DSM-project/memory"
	"DSM-project/treadmarks"
	"fmt"
)

func setupTreadMarksStruct1(nrProcs, memsize, pagebytesize, nrlocks, nrbarriers int) *treadmarks.TreadMarks {
	vm1 := memory.NewVmem(memsize, pagebytesize)
	tm1 := treadmarks.NewTreadMarks(vm1, nrProcs, nrlocks, nrbarriers)
	return tm1
}

//Benchmarks.SortedIntBenchmark(1, 1000, true, 8388608, 524288, 10)
func SortedIntTMBenchmark(nrProcs int, batchSize int, isManager bool, N int, Bmax int, Imax int) {
	//First we do setup.

	rand := NewRandom()
	tm := setupTreadMarksStruct1(nrProcs, N*8, 8, 2, 3)

	defer tm.Shutdown()
	key := func(i int) int { return (i + 1) * 4 }

	if isManager {
		fmt.Println("I'm a manager.")
		tm.Startup()
	} else {
		tm.Join("localhost:2000")
		fmt.Println("joined with id: ", tm.ProcId)
	}
	fmt.Println("Making numbers")
	K := make([]int, N)
	for i := 0; i < N; i++ {
		K[i] = int(float64(Bmax) * ((rand.Next() + rand.Next() + rand.Next() + rand.Next()) / 4))
	}
	fmt.Println("Reached first barrier")
	tm.Barrier(0)
	fmt.Println("Crossed first barrier.")
	for i := 1; i <= Imax; i++ {
		fmt.Println("Running iteration number ", i, ".")
		K[i] = i
		K[i+Imax] = Bmax - i
		//Calculate the order of every entry in the interval I am responsible for.
		start := 0
		for {
			tm.AcquireLock(0)
			start = tm.ReadInt(0)
			end := Min(start+batchSize, N)
			tm.WriteInt(0, end)
			tm.ReleaseLock(0)
			if start >= N {
				break
			}
			var kj int
			var rj int
			for j := start; j < end; j++ {
				if j%100 == 0 {
					fmt.Println("Progress: ", j)
				}
				//
				kj = K[j]
				rj = 0
				for l, kl := range K {
					if kj < kl || (kj == kl && l < j) {
						rj++
					}
				}
				tm.WriteInt(key(j), rj)
			}
		}
		tm.Barrier(1)
		//Partial verification
		if isManager && N > 8388607 {
			if N > 8388607 {
				fmt.Println("Manager is doing partial verification.")
				fmt.Println("First value is ", tm.ReadInt(key(2112377)), "\n -- it should be ", 104+i)
				fmt.Println("Second value is  ", tm.ReadInt(key(662041)), "\n -- it should be ", 17523+i)
				fmt.Println("Third value is  ", tm.ReadInt(key(5336171)), "\n -- it should be ", 123928+i)
				fmt.Println("Fourth value is  ", tm.ReadInt(key(3642833)), "\n -- it should be ", 8288932-i)
				fmt.Println("Fifth value is  ", tm.ReadInt(key(4250760)), "\n -- it should be ", 8388264-i)
			}
			fmt.Println("Manager is resetting the startpoint.")
			tm.WriteInt(0, 0)
		}
		tm.Barrier(2)

	}
	sorted := make([]int, N)
	for i := 0; i < N; i++ {
		sorted[tm.ReadInt(key(i))] = K[i]
	}
	last := 0
	for _, v := range sorted {
		if last > v {
			fmt.Println("We had something that wasnt sorted correctly.")
		}
	}

}
