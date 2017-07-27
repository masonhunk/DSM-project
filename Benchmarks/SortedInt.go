package Benchmarks

import (
	"DSM-project/memory"
	"DSM-project/treadmarks"
	"encoding/binary"
	"fmt"
	"math"

	"DSM-project/multiview"
	"io/ioutil"
	"log"
)

func setupTreadMarksStruct1(nrProcs, memsize, pagebytesize, nrlocks, nrbarriers int) *treadmarks.TreadMarks {
	vm1 := memory.NewVmem(memsize, pagebytesize)
	tm1 := treadmarks.NewTreadMarks(vm1, nrProcs, nrlocks, nrbarriers)
	return tm1
}

func SortedIntMVBenchmark(nrProcs int, batchSize int, isManager bool, N int, Bmax int32, Imax int) {
	log.SetOutput(ioutil.Discard)
	mv := multiview.NewMultiView()
	rand := NewRandom()
	address := make([]int, N)
	pagebytesize := 8
	memsize := (((N+1)*4)/pagebytesize + 1) * pagebytesize
	prog := N*4 + ((N%(pagebytesize/4))+1)*memsize
	if isManager {

		mv.Initialize(memsize, pagebytesize, nrProcs)
		fmt.Println("I am the manager.")
		for i := range address {
			address[i], _ = mv.Malloc(4)
		}
		prog, _ = mv.Malloc(4)
	} else {
		mv.Join(memsize, pagebytesize)
		for i := range address {
			address[i] = i*4 + ((i%(mv.GetPageSize()/4))+1)*memsize
		}

	}
	K := make([]int32, N)
	for i := 0; i < N; i++ {
		K[i] = int32(float64(Bmax) * ((rand.Next() + rand.Next() + rand.Next() + rand.Next()) / 4))
	}
	mv.Barrier(0)
	for i := 1; i <= Imax; i++ {
		fmt.Println("Starting iteration ", i)
		K[i] = int32(i)
		K[i+Imax] = Bmax - int32(i)
		//Calculate the order of every entry in the interval I am responsible for.
		start := 0
		var end int
		for {
			mv.Lock(0)
			start = mv.ReadInt(prog)
			end = Min(start+batchSize, N)
			mv.WriteInt(prog, end)
			mv.Release(0)
			if start >= N {
				break
			}
			var kj int32
			var rj int
			for j := start; j < end; j++ {
				//
				kj = K[j]
				rj = 0
				for l, kl := range K {
					if kl < kj || (kj == kl && l < j) {
						rj++
					}
				}
				mv.WriteInt(address[j], rj)
			}
		}
		mv.Barrier(1)
		//Partial verification
		if isManager {
			if N > 8388607 {
				fmt.Println("Manager is doing partial verification.")
				fmt.Println("First value is ", mv.ReadInt(address[2112377]), "\n -- it should be ", 104+i)
				fmt.Println("Second value is  ", mv.ReadInt(address[662041]), "\n -- it should be ", 17523+i)
				fmt.Println("Third value is  ", mv.ReadInt(address[5336171]), "\n -- it should be ", 123928+i)
				fmt.Println("Fourth value is  ", mv.ReadInt(address[3642833]), "\n -- it should be ", 8288932-i)
				fmt.Println("Fifth value is  ", mv.ReadInt(address[4250760]), "\n -- it should be ", 8388264-i)
			}
			mv.WriteInt(prog, 0)

		}
		mv.Barrier(2)

	}
	sorted := make([]int32, N)
	for i := 0; i < N; i++ {
		sorted[mv.ReadInt(address[i])] = K[i]
	}
	var last int32 = 0
	x := 0
	for _, v := range sorted {
		if last > v {
			//fmt.Println("We had something that wasnt sorted correctly.")
			x++
		}
		last = v
	}
	fmt.Println("We had ", x, " things that wasnt sorted right.")
	fmt.Println(sorted)
	defer mv.Shutdown()

}

//Benchmarks.SortedIntBenchmark(1, 1000, true, 8388608, 524288, 10)
func SortedIntTMBenchmark(nrProcs int, batchSize int, isManager bool, N int, Bmax int32, Imax int) {
	//First we do setup.
	log.SetOutput(ioutil.Discard)
	rand := NewRandom()
	pagebytesize := 8
	tm := setupTreadMarksStruct1(nrProcs, (((N+1)*4)/pagebytesize+1)*pagebytesize, 8, 2, 3)
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
	K := make([]int32, N)
	for i := 0; i < N; i++ {
		K[i] = int32(float64(Bmax) * ((rand.Next() + rand.Next() + rand.Next() + rand.Next()) / 4))
	}
	for i := 1; i <= Imax; i++ {
		fmt.Println("Starting iteration: ", i)
		K[i] = int32(i)
		K[i+Imax] = Bmax - int32(i)
		//Calculate the order of every entry in the interval I am responsible for.
		start := 0
		var end int
		for {
			tm.AcquireLock(0)
			start = tm.ReadInt(0)
			end = Min(start+batchSize, N)
			tm.WriteInt(0, end)
			tm.ReleaseLock(0)
			if start >= N {
				break
			}
			var kj int32
			var rj int
			for j := start; j < end; j++ {
				//
				kj = K[j]
				rj = 0
				for l, kl := range K {
					if kl < kj || (kj == kl && l < j) {
						if kj == 22 || kj == 20 {
						}

						rj++
					}
				}
				tm.WriteInt(key(j), rj)
			}
		}
		tm.Barrier(1)
		//Partial verification
		if isManager {
			if N > 8388607 {
				fmt.Println("Manager is doing partial verification.")
				fmt.Println("First value is ", tm.ReadInt(key(2112377)), "\n -- it should be ", 104+i)
				fmt.Println("Second value is  ", tm.ReadInt(key(662041)), "\n -- it should be ", 17523+i)
				fmt.Println("Third value is  ", tm.ReadInt(key(5336171)), "\n -- it should be ", 123928+i)
				fmt.Println("Fourth value is  ", tm.ReadInt(key(3642833)), "\n -- it should be ", 8288932-i)
				fmt.Println("Fifth value is  ", tm.ReadInt(key(4250760)), "\n -- it should be ", 8388264-i)
			}
			tm.WriteInt(0, 0)

		}
		tm.Barrier(2)

	}
	sorted := make([]int32, N)
	for i := 0; i < N; i++ {
		sorted[tm.ReadInt(key(i))] = K[i]
	}
	var last int32 = 0
	x := 0
	for _, v := range sorted {
		if last > v {
			x++
		}
		last = v
	}
	fmt.Println("We had ", x, " things that wasn't sorted right.")
	fmt.Println(sorted)

}

func intToBytes(i uint32) []byte {
	result := make([]byte, binary.Size(i))
	binary.PutUvarint(result, uint64(i))
	return result
}

func bytesToInt(bytes []byte) uint32 {
	n, _ := binary.Uvarint(bytes)
	return uint32(n)
}

type Random struct {
	x float64
}

func NewRandom() *Random {
	r := Random{x: 314159265}
	return &r
}

func (r *Random) Next() float64 {
	result := r.x * math.Pow(2, -46)
	r.x = math.Mod(r.x*1220703125, 70368744177664)
	return result
}

// Here is some utility stuff
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
