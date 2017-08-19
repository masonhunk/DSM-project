package datastructure

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"DSM-project/treadmarks"
	"fmt"
)

func TestPageArrayEntry_CopySet(t *testing.T) {
	pe := NewPageArrayEntry(3)
	assert.Len(t, pe.GetCopySet(), 1)
	assert.Equal(t, byte(0), pe.GetCopySet()[0])
	pe.AddToCopySet(byte(1))
	assert.Len(t, pe.GetCopySet(), 2)
	assert.Equal(t, byte(0), pe.GetCopySet()[0])
	assert.Equal(t, byte(1), pe.GetCopySet()[1])
}

func TestPageArrayEntry_AddGetWriteNoticeRecord(t *testing.T) {
	pe := NewPageArrayEntry(3)
	ts := treadmarks.NewTimestamp(3)
	id := WriteNoticeId(1, byte(1), ts)
	assert.Nil(t, pe.GetWriteNoticeRecordById(id))
	var w WriteNoticeRecordInterface = NewWriteNoticeRecord(1, byte(1), ts, nil)
	assert.True(t,pe.AddWriteNoticeRecord(byte(1), w))
	assert.Equal(t, w.GetId(), pe.GetWriteNoticeRecordById(id).GetId())
	assert.Equal(t, w.GetId(), pe.GetWriteNoticeRecord(1, byte(1), ts).GetId())

	assert.Nil(t, pe.GetWriteNoticeRecordById(id).GetDiff())
	var diffs Diff = make(map[int]byte)
	diffs[1] = byte(2)
	pe.GetWriteNoticeRecordById(id).SetDiff(diffs)
	assert.Equal(t, diffs, pe.GetWriteNoticeRecordById(WriteNoticeId(1, byte(1), ts)).GetDiff())
	assert.Equal(t, byte(2), pe.GetWriteNoticeRecordById(WriteNoticeId(1, byte(1), ts)).GetDiff()[1])

}

func TestPageArrayEntry_HasCopy(t *testing.T) {
	pe := NewPageArrayEntry(3)
	assert.False(t, pe.GetHasCopy())
	pe.SetHasCopy(true)
	assert.True(t, pe.GetHasCopy())
}

//Test if map keeps being consistent, even when we have expanded the underlying array.
func TestPageArrayEntry_wnrMap(t *testing.T){
	pe := NewPageArrayEntry(2)
	ts := treadmarks.NewTimestamp(2)
	var w0 WriteNoticeRecordInterface = NewWriteNoticeRecord(1, byte(1), ts, nil)
	assert.True(t, pe.AddWriteNoticeRecord(byte(1), w0))
	assert.Equal(t, fmt.Sprint("int_", byte(1), ts), pe.GetWriteNoticeRecord(1, byte(1), ts).GetIntervalId())
	var w WriteNoticeRecordInterface
	n := 0
	i := 1
	tst := ts
	for n < 4{ //Keep adding till we have expanded the list 3 times.
		tst = tst.Increment(byte(1))
		w = NewWriteNoticeRecord(1, byte(1), tst, nil)
		if pe.AddWriteNoticeRecord(byte(1), w){
			n++
		}
		assert.NotNil(t, pe.GetWriteNoticeRecord(1, byte(1), tst))
		i++
	}
	assert.Equal(t, treadmarks.Timestamp{0,0}, ts)
	tst = ts
	for j := 1; j < i; j++{
		tst = tst.Increment(byte(1))
		//Check if all the entries are correct when we get them.
		assert.NotNil(t, pe.GetWriteNoticeRecord(1, byte(1), tst))
		assert.Equal(t, fmt.Sprint("int_", byte(1), tst), pe.GetWriteNoticeRecord(1, byte(1), tst).GetIntervalId())
	}
	diff := Diff{3:byte(3)}
	w0.SetDiff(diff)
	//Test if references are still correct and everything.
	assert.Equal(t, diff, pe.GetWriteNoticeRecord(1, byte(1), ts).GetDiff())
}

func TestPageArrayEntry_GetMissingDiffs(t *testing.T) {
	pe := NewPageArrayEntry(3)
	assert.Nil(t, pe.GetMissingDiffs())
	ts1 := treadmarks.NewTimestamp(3)
	ts2 := ts1.Increment(byte(2))
	ts3 := ts2.Increment(byte(2))
	var w1 WriteNoticeRecordInterface = NewWriteNoticeRecord(1, byte(1), ts1, nil)
	var w2 WriteNoticeRecordInterface = NewWriteNoticeRecord(1, byte(1), ts2, nil)
	var w3 WriteNoticeRecordInterface = NewWriteNoticeRecord(1, byte(1), ts3, nil)
	pe.AddWriteNoticeRecord(byte(1), w1)
	assert.Len(t, pe.GetMissingDiffs(), 1)
	assert.Equal(t, DiffDescription{byte(1), ts1, nil}, pe.GetMissingDiffs()[0])
	pe.AddWriteNoticeRecord(byte(1), w2)
	assert.Len(t, pe.GetMissingDiffs(), 2)
	assert.Equal(t, DiffDescription{byte(1), ts1, nil}, pe.GetMissingDiffs()[0])
	assert.Equal(t, DiffDescription{byte(1), ts2, nil}, pe.GetMissingDiffs()[1])
	pe.AddWriteNoticeRecord(byte(2), w3)
	assert.Len(t, pe.GetMissingDiffs(), 3)
	assert.Equal(t, DiffDescription{byte(1), ts1, nil}, pe.GetMissingDiffs()[0])
	assert.Equal(t, DiffDescription{byte(1), ts2, nil}, pe.GetMissingDiffs()[1])
	assert.Equal(t, DiffDescription{byte(2), ts3, nil}, pe.GetMissingDiffs()[2])
	w1.SetDiff(Diff{1:byte(2)})
	assert.Len(t, pe.GetMissingDiffs(), 2)
	assert.Equal(t, DiffDescription{byte(1), ts2, nil}, pe.GetMissingDiffs()[0])
	assert.Equal(t, DiffDescription{byte(2), ts3, nil}, pe.GetMissingDiffs()[1])
	w3.SetDiff(Diff{2:byte(1)})
	assert.Len(t, pe.GetMissingDiffs(), 1)
	assert.Equal(t, DiffDescription{byte(1), ts2, nil}, pe.GetMissingDiffs()[0])

	pe = NewPageArrayEntry(3)
	assert.Nil(t, pe.GetMissingDiffs())
	ts1 = treadmarks.NewTimestamp(3)
	ts2 = ts1.Increment(byte(1))
	ts3 = ts1.Increment(byte(2))
	w1 = NewWriteNoticeRecord(1, byte(1), ts1, nil)
	w2 = NewWriteNoticeRecord(1, byte(1), ts2, nil)
	w3 = NewWriteNoticeRecord(1, byte(1), ts3, nil)
	pe.AddWriteNoticeRecord(byte(1), w1)
	pe.AddWriteNoticeRecord(byte(1), w2)
	pe.AddWriteNoticeRecord(byte(1), w3)
	assert.Len(t, pe.GetMissingDiffs(), 3)
	assert.Equal(t, DiffDescription{byte(1), ts1, nil}, pe.GetMissingDiffs()[0])
	assert.Equal(t, DiffDescription{byte(1), ts2, nil}, pe.GetMissingDiffs()[1])
	assert.Equal(t, DiffDescription{byte(1), ts3, nil}, pe.GetMissingDiffs()[2])
}