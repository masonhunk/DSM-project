package main

import (
	"DSM-project/Benchmarks"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var benchmark = flag.String("benchmark", "default", "choose benchmark algorithm")
var nrprocs = flag.Int("hosts", 1, "choose number of hosts.")
var port = flag.Int("port", 2000, "Choose port.")
var manager = flag.Bool("manager", true, "choose if instance is manager.")

func main() {
	flag.Parse()
	var cpuprofFile io.Writer
	if *cpuprofile == "" {
		cpuname := *benchmark
		cpuprofFile = setupCPUProf(cpuname)
	} else {
		cpuprofFile = setupCPUProf(*cpuprofile)
	}

	defer pprof.StopCPUProfile()
	fmt.Println(*benchmark)
	log.SetOutput(ioutil.Discard)
	switch *benchmark {
	case "ModuloMultMW":
		wg := sync.WaitGroup{}
		wg.Add(1)
		pageSize := 4096
		nrOfInts := 4096 * 1000000
		batchSize := 10000 * 4096 // nr of ints in batch
		Benchmarks.ParallelSumMW(batchSize, nrOfInts, *nrprocs, *manager, pageSize, &wg, cpuprofFile)
	case "JacobiTM":
		wg := sync.WaitGroup{}
		wg.Add(1)
		matrixsize := 64
		Benchmarks.JacobiProgramTreadMarks(matrixsize, 8, *nrprocs, *manager, &wg)
	case "JacobiMW":
		wg := sync.WaitGroup{}
		wg.Add(1)
		matrixsize := 1536
		Benchmarks.JacobiProgramMultiView(matrixsize, 20, *nrprocs, *manager, 4096, &wg, cpuprofFile)
	case "SortedIntTM":
		Benchmarks.SortedIntTMBenchmark(nil, *port, *nrprocs, 1000, *manager, 8388608, 524288, 10)
	case "SortedIntMW":
		batchSize := 1000
		N := 38860
		var Bmax int32 = 524288
		Imax := 10
		Benchmarks.SortedIntMVBenchmark(*nrprocs, batchSize, *manager, N, Bmax, Imax)
	case "SyncOpsCostMW":
		Benchmarks.TestSynchronizedWritesMW(*nrprocs, 10000, cpuprofFile)
	case "NonSyncOpsCostMW":
		Benchmarks.TestNonSynchronizedReadWritesMW(200000000, cpuprofFile)
	case "barrMW":
		Benchmarks.TestBarrierTimeMW(100000, *nrprocs, nil)
	case "locksMW":
		Benchmarks.TestLockMW(2000000, cpuprofFile)
	case "barrTM":
		Benchmarks.TestBarrierTimeTM(100000, *nrprocs, cpuprofFile)
	case "locksTM":
		Benchmarks.TestLockTM(2000000, cpuprofFile)
	case "SyncOpsCostTM":
		Benchmarks.TestSynchronizedReadsWritesTM(5000, cpuprofFile)
	case "NonSyncOpsCostTM":
		Benchmarks.TestNonSynchronizedReadWritesTM(100000000, cpuprofFile)
	default:
		group := new(sync.WaitGroup)
		go Benchmarks.SortedIntTMBenchmark(group, 1000, 2, 100, true, 10000, 524288, 10)
		time.Sleep(time.Millisecond * 500)
		Benchmarks.SortedIntTMBenchmark(group, 1001, 2, 100, false, 10000, 524288, 10)
	}

	if *memprofile == "" {
		memname := *benchmark
		setupMemProf(memname)
	} else {
		setupMemProf(*memprofile)
	}

}

func setupCPUProf(filename string) io.Writer {
	filename = "BenchmarkResults/" + filename
	i := 0
	for {
		_, err := os.Stat(fmt.Sprintf("%s%s%d%s", filename, "_", i, ".prof"))
		if os.IsNotExist(err) {
			break
		}
		i++
	}
	f, err := os.Create(fmt.Sprintf("%s%s%d%s", filename, "_", i, ".prof"))
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	return f
}

func setupMemProf(filename string) {
	filename = "BenchmarkResults/" + filename
	i := 0
	for {
		_, err := os.Stat(fmt.Sprintf("%s%s%d%s", filename, "_", i, ".mprof"))
		if os.IsNotExist(err) {
			break
		}
		i++
	}
	f, err := os.Create(fmt.Sprintf("%s%s%d%s", filename, "_", i, ".mprof"))
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	f.Close()
}
