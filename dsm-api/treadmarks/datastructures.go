package treadmarks

import (
	"github.com/Workiva/go-datastructures/bitarray"
	"sync"
)

type data []byte

type IntervalRecord struct {
	Owner     uint8
	Timestamp Timestamp
	Pages     []int16
}

type WritenoticeRecord struct {
	Owner uint8
	Timestamp Timestamp
	Diff      map[int]byte
}

type accessControl struct {
	read_only bitarray.BitArray
	write     bitarray.BitArray
}

type pageArrayEntry struct {
	hasMissingDiffs bool
	hasCopy      bool
	copySet      []uint8
	writenotices [][]WritenoticeRecord
}

func NewPageArray(nrPages int, nrProcs uint8) []*pageArrayEntry {
	array := make([]*pageArrayEntry, nrPages)
	for i := range array {
		array[i] = NewPageArrayEntry(nrProcs)
	}
	return array
}

func NewPageArrayEntry(nrProcs uint8) *pageArrayEntry {
	wnl := make([][]WritenoticeRecord, nrProcs)
	for j := range wnl {
		wnl[j] = make([]WritenoticeRecord, 0)
	}
	entry := &pageArrayEntry{
		hasCopy:      false,
		copySet:      []uint8{0},
		writenotices: wnl,
	}
	return entry
}

func NewProcArray(nrProcs uint8) [][]IntervalRecord {
	procarray := make([][]IntervalRecord, nrProcs)
	for i := range procarray {
		procarray[i] = make([]IntervalRecord, 0)
	}
	return procarray
}

type lock struct {
	sync.Locker
	locked        bool
	haveToken     bool
	last          uint8
	nextId        uint8
	nextTimestamp Timestamp
}
