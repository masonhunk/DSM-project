package Benchmarks

import (
	"DSM-project/dsm-api"
	"DSM-project/dsm-api/treadmarks"
	"DSM-project/multiview"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"runtime/pprof"
	"sync"
	"time"
)

func SortedIntMVBenchmark(nrProcs int, batchSize int, isManager bool, N int, Bmax int32, Imax int, pprofFile io.Writer) {
	log.SetOutput(ioutil.Discard)
	mv := multiview.NewMultiView()

	rand := NewRandom()
	address := make([]int, N)
	pagebytesize := 128
	memsize := (((N+1)*4)/pagebytesize + 1) * pagebytesize
	prog := N*4 + ((N%(pagebytesize/4))+1)*memsize
	if isManager {
		mv.Initialize(memsize, pagebytesize, nrProcs)
		fmt.Println("I am the manager.")
		allocs := make([]int, len(address))
		for i := range address {
			allocs[i] = 4
			//fmt.Println("manager address", i, ":",address[i])
		}
		address, _ = mv.MultiMalloc(allocs)
		prog, _ = mv.Malloc(4)
		mv.Barrier(3)
	} else {
		mv.Join(memsize, pagebytesize)
		mv.Barrier(3)
		for i := range address { /*
				//address[i] = i*4 + ((i%(mv.GetPageSize()/4))+1)*memsize
				k := i * N * 4
				l := k + j*4
				q := (i + j) % (mw.GetPageSize() / float32_BYTE_LENGTH)
				memsize := M * N * float32_BYTE_LENGTH
				vAddr := (q+1)*memsize + l*/

			k := 4 * i
			q := 0
			if 4%pagebytesize != 0 {
				q = i % max((pagebytesize/4), 2)
			}
			vAddr := (q+1)*mv.GetMemoryByteSize() + k
			//fmt.Println("host", mv.Id, "address", i, ":", vAddr)
			address[i] = vAddr
			mv.ReadInt(vAddr)
		}

	}
	mv.SetShouldLogNetwork(true)
	K := make([]int32, N)
	for i := 0; i < N; i++ {
		K[i] = int32(float64(Bmax) * ((rand.Next() + rand.Next() + rand.Next() + rand.Next()) / 4))
	}

	var startTime time.Time
	if isManager {
		startTime = time.Now()
	}
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	batchesInFirstIteration := make([]int, 0)
	mv.Barrier(0)
	for i := 1; i <= Imax; i++ {
		fmt.Println("Starting iteration ", i)
		K[i] = int32(i)
		K[i+Imax] = Bmax - int32(i)
		//Calculate the order of every entry in the interval I am responsible for.
		start := 0
		batchIndex := 0
		var end int
		for {
			if i == 1 {
				mv.Lock(0)
				start = mv.ReadInt(prog)
				batchesInFirstIteration = append(batchesInFirstIteration, start)
				end = Min(start+batchSize, N)
				mv.WriteInt(prog, end)
				mv.Release(0)

			} else {
				start = batchesInFirstIteration[batchIndex]
				batchIndex++
				end = Min(start+batchSize, N)

			}
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
		fmt.Println("ending iteration", i)
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

	if isManager {
		endTime := time.Now()
		diff := endTime.Sub(startTime)
		fmt.Println("execution time:", diff.String())
	}

	fmt.Println("We had ", x, " things that wasnt sorted right.")
	if isManager {
		mv.Barrier(4)
		mv.Shutdown()
	} else {
		mv.Barrier(4)
		mv.Leave()
	}

}

//Benchmarks.SortedIntBenchmark(1, 1000, true, 8388608, 524288, 10)
func SortedIntTMBenchmark(group *sync.WaitGroup, port, nrProcs, batchSize int, isManager bool, N int, Bmax int32, Imax int, pprofFile io.Writer) {
	//First we do setup.
	//log.SetOutput(ioutil.Discard)
	//setupTreadMarksStruct1(nrProcs, (((N+1)*4)/pagebytesize+1)*pagebytesize, 8, 2, 4)
	if group != nil {
		group.Add(1)
	}
	rand := NewRandom()
	pagebytesize := 4096
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

		tm.Join("localhost", 2000)
	}

	fmt.Println("Making numbers")
	K := make([]int32, N)
	for i := 0; i < N; i++ {
		K[i] = int32(float64(Bmax) * ((rand.Next() + rand.Next() + rand.Next() + rand.Next()) / 4))
	}

	var startTime time.Time
	if isManager {
		startTime = time.Now()
	}
	if pprofFile != nil {
		if err := pprof.StartCPUProfile(pprofFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	tm.SetLogging(true)
	fmt.Println(tm.GetId(), " at barrier 0")
	tm.Barrier(0)
	fmt.Println(tm.GetId(), " passed barrier 0")
	batchesInFirstIteration := make([]int, 0)
	for i := 1; i <= Imax; i++ {
		fmt.Println(tm.GetId(), "Starting iteration: ", i)
		K[i] = int32(i)
		K[i+Imax] = Bmax - int32(i)
		//Calculate the order of every entry in the interval I am responsible for.
		start := 0
		var end int
		batchIndex := 0
		for {
			if i == 1 {
				tm.AcquireLock(0)
				start = readInt(tm, 0)
				batchesInFirstIteration = append(batchesInFirstIteration, start)
				writeInt(tm, 0, end)
				tm.ReleaseLock(0)
			} else {
				start = batchesInFirstIteration[batchIndex]
				batchIndex++
			}

			if start >= N {
				tm.ReleaseLock(0)
				break
			}
			end = Min(start+batchSize, N)
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
			writeInt(tm, 0, 0)

		}
		tm.Barrier(2)

	}
	tm.Barrier(3)
	sorted := make([]int32, N)
	for i := 0; i < N; i++ {
		x := readInt(tm, key(i))
		//fmt.Println(i, " - ", K[i], " - ", x, " - ", N)
		if len(sorted) <= x {
			sorted = append(sorted, make([]int32, len(sorted))...)
		}
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
	fmt.Println(tm.GetId(), " at barrier 3")
	tm.Barrier(3)
	fmt.Println(tm.GetId(), " passed barrier 3")
	if isManager {
		endTime := time.Now()
		diff := endTime.Sub(startTime)
		fmt.Println("execution time:", diff.String())
	}

	fmt.Println("We had ", x, " things that wasn't sorted right.")
	if group != nil {
		group.Done()
		group.Wait()
	}

}

func readInt(dsm dsm_api.DSMApiInterface, addr int) int {
	bInt, _ := dsm.ReadBytes(addr, 4)
	result, n := binary.Varint(bInt)
	if n == 0 {
		panic("buf too small for varint.")
	}
	return int(result)
}

func writeInt(dsm dsm_api.DSMApiInterface, addr, input int) {
	bInt := make([]byte, 4)
	binary.PutVarint(bInt, int64(input))
	dsm.WriteBytes(addr, bInt)
}

func readInt64(dsm dsm_api.DSMApiInterface, addr int) int64 {
	bInt, _ := dsm.ReadBytes(addr, 8)
	result, n := binary.Varint(bInt)
	if n == 0 {
		panic("buf too small for varint.")
	}
	return result
}

func writeInt64(dsm dsm_api.DSMApiInterface, addr int, input int64) {
	bInt := make([]byte, 8)
	binary.PutVarint(bInt, input)
	dsm.WriteBytes(addr, bInt)
}
