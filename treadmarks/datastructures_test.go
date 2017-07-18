package treadmarks

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPageArray(t *testing.T) {
	//vm := memory.NewVmem(1024, 64)
	pageArray := make(PageArray)
	wn1 := WriteNoticeRecord{Diff: nil, Interval: nil, NextRecord: nil, PrevRecord: nil}
	wn2 := WriteNoticeRecord{Diff: nil, Interval: nil, NextRecord: nil, PrevRecord: &wn1}
	wn1.NextRecord = &wn2
	pageArray[1] = PageArrayEntry{
		CopySet: []int{1, 2},
		ProcArr: map[byte]*WriteNoticeRecord{0:&wn1},
	}
	assert.Equal(t, wn1, *pageArray[1].ProcArr[0])
	assert.Nil(t, pageArray[1].ProcArr[0].PrevRecord)
	assert.Equal(t, wn2, *pageArray[1].ProcArr[0].NextRecord)
	assert.Nil(t, pageArray[1].ProcArr[0].NextRecord.NextRecord)

}


func TestPageArrayEntry_AddWriteNotice(t *testing.T) {
	pe := NewPageArrayEntry()
	wn1 := pe.AddWriteNotice(byte(0))
	assert.Equal(t, wn1, pe.ProcArr[0])
	wn2 := pe.AddWriteNotice(byte(0))
	assert.Equal(t, wn2, pe.ProcArr[0])
	assert.Equal(t, wn1, pe.ProcArr[0].NextRecord)
	assert.Equal(t, wn2, pe.ProcArr[0].NextRecord.PrevRecord)
}

func TestPair_AppendIntervalRecord(t *testing.T) {
	pair := new(Pair)
	v1 := NewVectorclock(2)
	v1.Increment(byte(1))
	ir1 := new(IntervalRecord)
	ir1.Timestamp = *v1
	v2 := NewVectorclock(2)
	v2.Increment(byte(0))
	v2.Increment(byte(0))
	ir2 := new(IntervalRecord)
	ir2.Timestamp = *v2
	pair.AppendIntervalRecord(ir1)
	car := pair.car.(*IntervalRecord)
	assert.Equal(t, uint(1), car.Timestamp.GetTick(byte(1)))
	pair.AppendIntervalRecord(ir2)
	car = pair.car.(*IntervalRecord)
	cdr := pair.cdr.(*IntervalRecord)
	assert.Equal(t, uint(1), car.Timestamp.GetTick(byte(1)))
	assert.Equal(t, uint(2), cdr.Timestamp.GetTick(byte(0)))
}


func TestPair_PrependIntervalRecord(t *testing.T) {
	pair := new(Pair)
	v1 := NewVectorclock(2)
	v1.Increment(byte(1))
	ir1 := new(IntervalRecord)
	ir1.Timestamp = *v1
	v2 := NewVectorclock(2)
	v2.Increment(byte(0))
	v2.Increment(byte(0))
	ir2 := new(IntervalRecord)
	ir2.Timestamp = *v2
	pair.PrependIntervalRecord(ir1)
	car := pair.car.(*IntervalRecord)
	assert.Equal(t, uint(1), car.Timestamp.GetTick(byte(1)))
	pair.PrependIntervalRecord(ir2)
	car = pair.car.(*IntervalRecord)
	cdr := pair.cdr.(*IntervalRecord)
	assert.Equal(t, uint(2), car.Timestamp.GetTick(byte(0)))
	assert.Equal(t, uint(1), cdr.Timestamp.GetTick(byte(1)))
}

