package treadmarks

import (
	"fmt"
	"strings"
)

type Timestamp []int

func NewTimestamp(nrProcs int) Timestamp{
	return make([]int, nrProcs)
}

func (t Timestamp) String() string {
	s := make([]string, len(t))
	for i := range t{
		s[i] = fmt.Sprint(t[i])
	}
	return "["+strings.Join(s, ",")+"]"
}

func (t Timestamp) Equals(o Timestamp) bool{
	if len(t) != len(o) {
		return false
	}
	for i := 0; i < len(t); i++{
		if t[i] != o[i] {
			return false
		}
	}
	return true
}

func (t Timestamp) Increment(procId byte) Timestamp {
	newT := make([]int, len(t))
	copy(newT, t)
	newT[int(procId)-1] = newT[int(procId)-1] +1
	return newT
}

func (t Timestamp) Less(o Timestamp) bool {
	if len(t) != len(o){
		panic("Tried to compare timestamps of different length!")
	}
	less := true
	notEqual := false

	for i := 0; i < len(t); i++{
		less = less && t[i] <= o[i]
		notEqual = notEqual || t[i] != o[i]
	}
	if !less && notEqual{
		for i := 0; i < len(t); i++{
			if t[i] == o[i] {
				continue
			} else {
				less = t[i] < o[i]
				break
			}
		}
	}
	return less && notEqual
}

type ByVal []Timestamp

func (a ByVal) Len() int           { return len(a) }
func (a ByVal) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByVal) Less(i, j int) bool { return a[i].Less(a[j]) }
