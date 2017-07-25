package Benchmarks

import (
	"DSM-project/memory"
	"DSM-project/treadmarks"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

func BenchmarkTreadMarks(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {

	}
}

func setupTreadMarksStruct(nrProcs, memsize, pagebytesize, nrlocks, nrbarriers int) *treadmarks.TreadMarks {
	vm1 := memory.NewVmem(memsize, pagebytesize)
	tm1 := treadmarks.NewTreadMarks(vm1, nrProcs, nrlocks, nrbarriers)
	return tm1
}

func JacobiProgramTreadMarks(nrProcs int, isManager bool) {
	const M = 1024
	const N = 1024
	const float32_BYTE_LENGTH = 4 //32 bits
	var privateArray [][]float32  //privateArray[M][N]
	privateArray = make([][]float32, M)
	for i := range privateArray {
		privateArray[i] = make([]float32, N)
	}
	host := setupTreadMarksStruct(nrProcs, M*N*float32_BYTE_LENGTH, 4096, 1, 3)
	if isManager {
		host.Startup()
	} else {
		host.Join("localhost:2000")
	}

}

func bytesToFloat32(bytes []byte) float32 {
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float
}

func float32ToBytes(float float32) []byte {
	bits := math.Float32bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint32(bytes, bits)
	return bytes
}
