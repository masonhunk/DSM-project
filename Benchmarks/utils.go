package Benchmarks

import (
	"DSM-project/memory"
	"DSM-project/treadmarks"
	"encoding/binary"
	"math"
	"DSM-project/dsm-api"
	"bytes"
)

func setupTreadMarksStruct(nrProcs, memsize, pagebytesize, nrlocks, nrbarriers int) *treadmarks.TreadMarks {
	vm1 := memory.NewVmem(memsize, pagebytesize)
	tm1 := treadmarks.NewTreadMarks(vm1, nrProcs, nrlocks, nrbarriers)
	return tm1
}

func bytesToFloat32(bytes []byte) float32 {
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float
}

func float32ToBytes(float float32) []byte {
	bits := math.Float32bits(float)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)
	return bytes
}

func readFloat(dsm dsm_api.DSMApiInterface,addr int) float64 {
	bInt, _ := dsm.ReadBytes(addr, 8)
	buf := bytes.NewBuffer(bInt)
	var result float64
	err := binary.Read(buf, binary.BigEndian, &result)
	if err != nil {
		panic(err.Error())
	}
	return result
}

func writeFloat(dsm dsm_api.DSMApiInterface, addr int, value float64) {
	buf := bytes.NewBuffer(make([]byte, 0, 8))
	binary.Write(buf, binary.BigEndian, value)
	if buf.Len() > 8 {
		panic("floats are too big!")
	}
	dsm.WriteBytes(addr, buf.Bytes())
}



func mergeArrays(a, b []int) []int {
	res := make([]int, len(a)+len(b))
	aHeadIndex := 0
	bHeadIndex := 0
	for i := range res {
		if aHeadIndex >= len(a) {
			res[i] = b[bHeadIndex]
			bHeadIndex++
			continue
		}
		if bHeadIndex >= len(b) {
			res[i] = a[aHeadIndex]
			aHeadIndex++
			continue
		}
		aHead := a[aHeadIndex]
		bHead := b[bHeadIndex]
		if aHead <= bHead {
			res[i] = aHead
			aHeadIndex++
		} else {
			res[i] = bHead
			bHeadIndex++
		}
	}
	return res
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

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}
