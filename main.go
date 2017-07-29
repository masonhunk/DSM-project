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
var nrprocs = flag.Int("hosts", 1, "choose number of hosts.")
var manager = flag.Bool("manager", true, "choose if instance is manager.")

func main() {
	flag.Parse()
	if *cpuprofile == "" {
		cpuname := *benchmark
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
		//Benchmarks.TestJacobiProgramMultiView()
	case "JacobiMW-manager":
		//wg := sync.WaitGroup{}
		//wg.Add(1)
		//Benchmarks.JacobiProgramMultiView(10, 2, true, 32, &wg)
	case "JacobiMW-host":
		wg := sync.WaitGroup{}
		wg.Add(1)
		//Benchmarks.JacobiProgramMultiView(10, 2, false, 32, &wg)
	case "SortedIntTM":
		Benchmarks.SortedIntTMBenchmark(*nrprocs, 1000, *manager, 8388608, 524288, 10)
	default:
		Benchmarks.TestMultipleSortedIntTM()
	}

	if *memprofile == "" {
		memname := *benchmark
		setupMemProf(memname)
	} else {
		setupMemProf(*memprofile)
	}

}

func setupCPUProf(filename string) {
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
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
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
