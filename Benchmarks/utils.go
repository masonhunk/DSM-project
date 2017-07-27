package Benchmarks

import (
	"encoding/binary"
	"math"
)

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
