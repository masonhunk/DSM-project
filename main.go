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
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var benchmark = flag.String("benchmark", "default", "choose benchmark algorithm")
var nrprocs = flag.Int("hosts", 1, "choose number of hosts.")
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
	case "ModuloMult-manager":
		wg := sync.WaitGroup{}
		wg.Add(1)
		pageSize := 4096
		nrOfInts := 4096 * 1000000
		batchSize := 10000 * 4096 // nr of ints in batch
		nrProcs := 4
		Benchmarks.ParallelSumMW(batchSize, nrOfInts, nrProcs, true, pageSize, &wg, cpuprofFile)
	case "ModuloMult-host":
		wg := sync.WaitGroup{}
		wg.Add(1)
		pageSize := 4096
		nrOfInts := 4096 * 1000000
		batchSize := 100 * 4096 // nr of ints in batch
		nrProcs := 4
		Benchmarks.ParallelSumMW(batchSize, nrOfInts, nrProcs, false, pageSize, &wg, nil)
	case "JacobiTM-manager":
		wg := sync.WaitGroup{}
		wg.Add(1)
		matrixsize := 64
		Benchmarks.JacobiProgramTreadMarks(matrixsize, 8, 1, true, &wg)
	case "JacobiTM-host":
		wg := sync.WaitGroup{}
		wg.Add(1)
		matrixsize := 64
		Benchmarks.JacobiProgramTreadMarks(matrixsize, 8, 1, false, &wg)
	case "JacobiMW-manager":
		wg := sync.WaitGroup{}
		wg.Add(1)
		matrixsize := 128
		Benchmarks.JacobiProgramMultiView(matrixsize, 10, 1, true, 4096, &wg)
	case "JacobiMW-host":
		wg := sync.WaitGroup{}
		wg.Add(1)
		matrixsize := 128
		Benchmarks.JacobiProgramMultiView(matrixsize, 10, 1, false, 4096, &wg)
	case "SortedIntTM":
		Benchmarks.SortedIntTMBenchmark(*nrprocs, 1000, *manager, 8388608, 524288, 10)
	default:
		//Benchmarks.TestMultipleSortedIntTM()
	}

	if *memprofile == "" {
		memname := *benchmark
		setupMemProf(memname)
	} else {
		setupMemProf(*memprofile)
	}

}

func setupCPUProf(filename string) io.Writer {
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
