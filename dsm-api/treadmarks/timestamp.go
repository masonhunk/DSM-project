package treadmarks

import (
	"fmt"
	"DSM-project/utils"
)

type Timestamp []int32

func NewTimestamp(nrProcs uint8) Timestamp {
	ts := Timestamp(make([]int32, nrProcs))
	return ts
}

func (t Timestamp) increment(procId uint8) Timestamp{
	ts := Timestamp(make([]int32, len(t)))
	copy(ts, t)
	ts[procId]++
	return ts
}

func (t Timestamp) close() Timestamp{
	ts := Timestamp(make([]int32, len(t)))
	copy(ts, t)
	return t
}

func (t Timestamp) covers(o Timestamp) bool {
	if len(t) != len(o) {
		panic(fmt.Sprint("Compared two timestamps of different length:\n", t, "\n", o))
	}
	for i := 0; i < len(t); i++{
		if t[i]<o[i] {
			return false
		}
	}
	return true
}

func (t Timestamp) equals(o Timestamp)bool {
	if len(t) != len(o) {
		panic(fmt.Sprint("Compared two timestamps of different length:\n", t, "\n", o))
	}
	for i := 0; i < len(t); i++{
		if t[i] != o[i] {
			return false
		}
	}
	return true
}

func (t Timestamp) merge(o Timestamp) Timestamp {
	newTs := Timestamp(make([]int32, len(t)))
	if len(t) != len(o) {
		panic(fmt.Sprint("Length were not matching", len(t), " != ",len(o)))
	}
	for i := range t {
		if t[i] > o[i] {
			newTs[i] = t[i]
		} else {
			newTs[i] = o[i]
		}
	}
	return newTs
}

func (t Timestamp) min(o Timestamp) Timestamp {
	newTs := Timestamp(make([]int32, len(t)))
	for i := range t {
		if t[i] < o[i] {
			newTs[i] = t[i]
		} else {
			newTs[i] = o[i]
		}
	}
	return newTs
}


func (t Timestamp) loadFromData(data []byte){
	for i := range t {
		t[i] = utils.BytesToInt32(data[i*4:i*4+4])
	}
}

func (t Timestamp) saveToData() []byte {
	data := make([]byte, 4*len(t))
	for i := range t {
		copy(data[i*4:i*4+3], utils.Int32ToBytes(t[i]))
	}
	return data
}