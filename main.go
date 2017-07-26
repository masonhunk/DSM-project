package main

import (
	"DSM-project/Benchmarks"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var benchmark = flag.String("benchmark", "default", "choose benchmark algorithm")

func main() {
	flag.Parse()
	if *cpuprofile == "" {
		cpuname := *benchmark + ".prof"
		setupCPUProf(cpuname)
	} else {
		setupCPUProf(*cpuprofile)
	}

	defer pprof.StopCPUProfile()
	fmt.Println(*benchmark)
	switch *benchmark {
	case "TSP":
		//Run tsp algorithm
	case "JacobiMW":
		Benchmarks.TestJacobiProgramMultiView()
	case "JacobiMW-manager":
		wg := sync.WaitGroup{}
		wg.Add(1)
		Benchmarks.JacobiProgramMultiView(10, 2, true, 32, &wg)
	case "JacobiMW-host":
		wg := sync.WaitGroup{}
		wg.Add(1)
		Benchmarks.JacobiProgramMultiView(10, 2, false, 32, &wg)
	default:

	}

	if *memprofile == "" {
		memname := *benchmark + ".mprof"
		setupMemProf(memname)
	} else {
		setupMemProf(*memprofile)
	}

}

func setupCPUProf(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
}

func setupMemProf(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	f.Close()
}

func mockBenchmark() {
	k := 2
	for i := 0; i < 1000000000; i++ {
		if k > 100 {
			k = mod(k)
		} else {
			k = mult(k)
		}
	}
}

func mult(i int) int {
	return i * 2
}

func mod(i int) int {
	return i % 1000
}
