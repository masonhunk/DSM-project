package Benchmarks

import (
	"DSM-project/multiview"
	"fmt"
	"io/ioutil"
	"log"
	"DSM-project/dsm-api/treadmarks"
	"DSM-project/dsm-api"
	"DSM-project/utils"
	"sync"
)


func SortedIntMVBenchmark( nrProcs int, batchSize int, isManager bool, N int, Bmax int32, Imax int) {
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
func SortedIntTMBenchmark(group *sync.WaitGroup, port, nrProcs, batchSize int, isManager bool, N int, Bmax int32, Imax int) {
	//First we do setup.
	//log.SetOutput(ioutil.Discard)
	//setupTreadMarksStruct1(nrProcs, (((N+1)*4)/pagebytesize+1)*pagebytesize, 8, 2, 4)
	if group != nil {
		group.Add(1)
	}
	rand := NewRandom()
	pagebytesize := 8
	tm, err := treadmarks.NewTreadmarksApi((((N+1)*4)/pagebytesize+1)*pagebytesize, pagebytesize, uint8(nrProcs), uint8(2), uint8(4))
	tm.Initialize(port)
	if err != nil {
		panic(err.Error())
	}
	defer tm.Shutdown()
	key := func(i int) int { return (i + 1) * 4 }

	if isManager {
		fmt.Println("I'm a manager.")
	} else {
		tm.Join("localhost", 1000)
	}

	fmt.Println("Making numbers")
	K := make([]int32, N)
	for i := 0; i < N; i++ {
		K[i] = int32(float64(Bmax) * ((rand.Next() + rand.Next() + rand.Next() + rand.Next()) / 4))
	}
	fmt.Println("Reached first barrier")
	tm.Barrier(0)
	fmt.Println("Crossed barrier")
	for i := 1; i <= Imax; i++ {
		fmt.Println("Starting iteration: ", i)
		K[i] = int32(i)
		K[i+Imax] = Bmax - int32(i)
		//Calculate the order of every entry in the interval I am responsible for.
		start := 0
		var end int
		for {
			tm.AcquireLock(0)
			start = readInt(tm, 0)
			end = Min(start+batchSize, N)
			writeInt(tm, 0, end)
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
						rj++
					}
				}
				writeInt(tm, key(j), rj)
				if err != nil {
					panic(err.Error())
				}
			}
		}
		tm.Barrier(1)
		//Partial verification
		if isManager {
			if N > 8388607 {
				fmt.Println("Manager is doing partial verification.")

				fmt.Println("First value is ", readInt(tm, key(2112377)), "\n -- it should be ", 104+i)
				fmt.Println("Second value is  ", readInt(tm, key(662041)), "\n -- it should be ", 17523+i)
				fmt.Println("Third value is  ", readInt(tm, key(5336171)), "\n -- it should be ", 123928+i)
				fmt.Println("Fourth value is  ", readInt(tm, key(3642833)), "\n -- it should be ", 8288932-i)
				fmt.Println("Fifth value is  ", readInt(tm, key(4250760)), "\n -- it should be ", 8388264-i)
			}
			writeInt(tm,0, 0)

		}
		tm.Barrier(2)

	}
	tm.Barrier(3)
	sorted := make([]int32, N)
	for i := 0; i < N; i++ {
		x := readInt(tm, key(i))
		fmt.Println(i, " - ", K[i], " - ", x, " - ", N)
		sorted[x] = K[i]
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
	if group != nil {
		group.Done()
		group.Wait()
	}

}

func readInt(dsm dsm_api.DSMApiInterface, addr int) int{
	bInt := make([]byte, 4)
	var err error
	for i := range bInt{
		bInt[i], err = dsm.Read(addr+i)
		if err != nil {
			panic(err.Error())
		}
	}
	return int(utils.BytesToInt32(bInt))
}

func writeInt(dsm dsm_api.DSMApiInterface, addr, input int) {
	bInt := utils.Int32ToBytes(int32(input))
	var err error
	for i := range bInt{
		err = dsm.Write(addr+i, bInt[i])
		if err != nil {
			panic(err.Error())
		}
	}
}
