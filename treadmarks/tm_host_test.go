package treadmarks

import (
	"DSM-project/memory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateDiff(t *testing.T) {
	original := []byte{0, 1, 2, 3, 4}
	new := []byte{0, 1, 3, 5, 4}
	assert.Equal(t, []Pair{Pair{2, byte(3)}, Pair{3, byte(5)}}, CreateDiff(original, new).Diffs)
}

func TestUpdateDatastructures(t *testing.T) {
	vm := memory.NewVmem(128, 8)
	tm := NewTreadMarks(vm, 1, 1, 1)
	tm.procId = 3
	tm.copyMap[0] = []byte{4, 4, 4, 4, 4, 4, 4, 4}
	tm.copyMap[1] = []byte{1, 1, 1, 1, 1, 1, 1, 1}
	tm.procArray = make(ProcArray, 4)
	tm.updateDatastructures()
	headWNRecord := tm.pageArray[0].ProcArr[3]
	headIntervalRecord := tm.procArray[3].car
	assert.Len(t, headIntervalRecord.(*IntervalRecord).WriteNotices, 2)
	assert.True(t, headWNRecord == headIntervalRecord.(*IntervalRecord).WriteNotices[0])
	assert.True(t, headIntervalRecord == headWNRecord.Interval)

	headWNRecord1 := tm.pageArray[1].ProcArr[3]

	assert.True(t, headWNRecord1 == headIntervalRecord.(*IntervalRecord).WriteNotices[1])
}

func TestPreprendInterval(t *testing.T) {
	p := Pair{nil, nil}
	a := &IntervalRecord{}
	b := &IntervalRecord{}
	c := &IntervalRecord{}
	p.PrependIntervalRecord(c)
	p.PrependIntervalRecord(b)
	p.PrependIntervalRecord(a)
	assert.True(t, p.car == a)
}
