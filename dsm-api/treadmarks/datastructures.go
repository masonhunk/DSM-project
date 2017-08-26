package treadmarks

import (
	"github.com/Workiva/go-datastructures/bitarray"
	"sync"
)

type data []byte

type IntervalRecord struct{
	owner     uint8
	timestamp Timestamp
	pages     []int16
}


type WritenoticeRecord struct{
	timestamp Timestamp
	diff      map[int]byte
}

type accessControl struct{
	read_only bitarray.BitArray
	write bitarray.BitArray
}

type pageArrayEntry struct{
	hasCopy bool
	copySet []uint8
	writenotices [][]WritenoticeRecord
}

type Lock struct{
	sync.Locker
	locked        bool
	held          bool
	last          uint8
	nextId        uint8
	nextTimestamp Timestamp
}
