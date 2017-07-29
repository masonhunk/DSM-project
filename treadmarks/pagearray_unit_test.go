package treadmarks

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWriteNoticeRecord_ToString(t *testing.T) {
	vc := *NewVectorclock(2)
	inter := &IntervalRecord{Timestamp: vc}
	wnr := &WriteNoticeRecord{Id: 2, pageNr: 2, Interval: inter}
	assert.Equal(t, "2 [0 0]", wnr.ToString())
	wnr.Id = 1
	assert.Equal(t, "2 [0 0]", wnr.ToString())
	wnr.pageNr = 1
	assert.Equal(t, "1 [0 0]", wnr.ToString())
	wnr.Interval = nil
	assert.Equal(t, "1 []", wnr.ToString())
}

func TestPageArrayEntry1_AddToCopyset(t *testing.T) {
	b := byte(1)
	pe := NewPageArrayEntry1(&b, 2, 1)
	assert.Contains(t, pe.copySet, byte(0))
	assert.Len(t, pe.copySet, 1)
	pe.AddToCopyset(byte(4))
	assert.Contains(t, pe.copySet, byte(0))
	assert.Contains(t, pe.copySet, byte(4))
	assert.Len(t, pe.copySet, 2)
}

func TestPageArrayEntry1_GetCopyset(t *testing.T) {
	b := byte(1)
	pe := NewPageArrayEntry1(&b, 2, 1)
	pe.AddToCopyset(byte(4))
	pe.AddToCopyset(byte(2))
	assert.Equal(t, []byte{0, 4, 2}, pe.copySet)
}

func TestPageArrayEntry1_HasAndSetCopy(t *testing.T) {
	b := byte(1)
	pe := NewPageArrayEntry1(&b, 2, 1)
	assert.False(t, pe.HasCopy())
	assert.False(t, pe.hasCopy)
	pe.SetHasCopy(true)
	assert.True(t, pe.HasCopy())
	assert.True(t, pe.hasCopy)
}

func TestPageArrayEntry1_AddWritenoticeRecord(t *testing.T) {
	b := byte(1)
	pe := NewPageArrayEntry1(&b, 2, 1)
	wnr1 := &WriteNoticeRecord{Id: 2}
	wnr2 := &WriteNoticeRecord{Id: 2}
	pe.AddWritenoticeRecord(byte(1), wnr1)
	assert.Equal(t, pe.writeNoticeArray[0][0].Id, wnr1.Id)
	pe.AddWritenoticeRecord(byte(1), wnr2)
	assert.Equal(t, pe.writeNoticeArray[0][0].Id, wnr2.Id)
	assert.Equal(t, pe.writeNoticeArray[0][1].Id, wnr1.Id)
	assert.Equal(t, wnr1.Id, pe.writeNoticeArrayVCLookup[wnr1.ToString()].Id)
	assert.Equal(t, wnr2.Id, pe.writeNoticeArrayVCLookup[wnr2.ToString()].Id)
}

func TestPageArrayEntry1_GetWriteNoticeRecord(t *testing.T) {
	b := byte(1)
	pe := NewPageArrayEntry1(&b, 2, 1)
	vc1 := *NewVectorclock(2)
	vc1.SetTick(byte(1), 2)
	vc2 := *NewVectorclock(2)
	int1 := &IntervalRecord{Timestamp: vc1}
	int2 := &IntervalRecord{Timestamp: vc2}
	wnr1 := &WriteNoticeRecord{Id: 1, Interval: int1}
	wnr2 := &WriteNoticeRecord{Id: 2, Interval: int2}
	pe.AddWritenoticeRecord(byte(1), wnr1)
	pe.AddWritenoticeRecord(byte(1), wnr2)
	assert.Equal(t, wnr1, pe.GetWriteNoticeRecord(byte(1), vc1))
	assert.Equal(t, wnr2, pe.GetWriteNoticeRecord(byte(1), vc2))
}

func TestPageArrayEntry1_SetDiff(t *testing.T) {
	b := byte(1)
	pe := NewPageArrayEntry1(&b, 2, 1)
	vc := NewVectorclock(2)
	vc.SetTick(byte(1), 2)
	inter := &IntervalRecord{Timestamp: *vc}
	wnr := &WriteNoticeRecord{Id: 1, Interval: inter}
	pe.AddWritenoticeRecord(byte(1), wnr)
	diff := DiffDescription{
		ProcId:    byte(1),
		Timestamp: *vc,
		Diff:      Diff{[]Pair{{byte(1), byte(2)}}},
	}
	pe.SetDiff(diff)
	assert.Equal(t, Pair{byte(1), byte(2)}, wnr.Diff.Diffs[0])
}

func TestPageArrayEntry1_SetDiffs(t *testing.T) {
	b := byte(1)
	pe := NewPageArrayEntry1(&b, 2, 1)
	vc1 := NewVectorclock(2)
	vc2 := NewVectorclock(2)
	vc1.SetTick(byte(1), 2)
	vc2.SetTick(byte(1), 1)
	int1 := &IntervalRecord{Timestamp: *vc1}
	int2 := &IntervalRecord{Timestamp: *vc2}
	wnr1 := &WriteNoticeRecord{Id: 1, Interval: int1}
	wnr2 := &WriteNoticeRecord{Id: 2, Interval: int2}
	pe.AddWritenoticeRecord(byte(1), wnr2)
	pe.AddWritenoticeRecord(byte(1), wnr1)
	/*diff1 := DiffDescription{
		ProcId:    byte(1),
		Timestamp: *vc1,
		Diff:      Diff{[]Pair{{byte(1), byte(2)}}},
	}
	diff2 := DiffDescription{
		ProcId:    byte(1),
		Timestamp: *vc2,
		Diff:      Diff{[]Pair{{byte(3), byte(4)}}},
	}
	//diffs := []DiffDescription{diff1, diff2}
	//pe.SetDiffs(diffs)
	assert.Equal(t, Pair{byte(1), byte(2)}, wnr1.Diff.Diffs[0])
	assert.Equal(t, Pair{byte(3), byte(4)}, wnr2.Diff.Diffs[0])
	*/
}

func TestPageArray1_GetCopyset(t *testing.T) {
	b := byte(1)
	p := NewPageArray1(&b, 2)
	assert.Equal(t, []byte{0}, p.GetCopyset(1))
	p.AddToCopyset(byte(1), 1)
	assert.Equal(t, []byte{0, 1}, p.GetCopyset(1))
	p.AddToCopyset(byte(4), 2)
	assert.Equal(t, []byte{0, 4}, p.GetCopyset(2))
}

func TestPageArray1_GetWritenoticeRecord(t *testing.T) {
	b := byte(1)
	p := NewPageArray1(&b, 2)
	vc1 := NewVectorclock(2)
	assert.Nil(t, p.GetWritenoticeRecord(byte(1), 1, *vc1))
	int1 := &IntervalRecord{Timestamp: *vc1}
	wnr1 := &WriteNoticeRecord{Id: 1, Interval: int1}
	p.AddWriteNoticeRecord(byte(1), 1, wnr1)
	assert.Equal(t, wnr1.Id, p.GetWritenoticeRecord(byte(1), 1, *vc1).Id)
	vc2 := NewVectorclock(2)
	vc2.Increment(byte(1))
	assert.Nil(t, p.GetWritenoticeRecord(byte(1), 1, *vc2))
	int2 := &IntervalRecord{Timestamp: *vc2}
	wnr2 := &WriteNoticeRecord{Id: 2, Interval: int2}
	p.AddWriteNoticeRecord(byte(1), 1, wnr2)
	assert.Equal(t, wnr1.Id, p.GetWritenoticeRecord(byte(1), 1, *vc1).Id)
	assert.Equal(t, wnr2.Id, p.GetWritenoticeRecord(byte(1), 1, *vc2).Id)
}

func TestPageArray1_MapWriteNotices(t *testing.T) {
	b := byte(1)
	p := NewPageArray1(&b, 2)
	vc1 := NewVectorclock(2)
	vc2 := NewVectorclock(2)
	vc2.Increment(byte(1))
	assert.Nil(t, p.GetWritenoticeRecord(byte(1), 1, *vc1))
	int1 := &IntervalRecord{Timestamp: *vc1}
	wnr1 := &WriteNoticeRecord{Id: 1, Interval: int1}
	int2 := &IntervalRecord{Timestamp: *vc2}
	wnr2 := &WriteNoticeRecord{Id: 2, Interval: int2}
	p.AddWriteNoticeRecord(byte(1), 1, wnr1)
	p.AddWriteNoticeRecord(byte(1), 1, wnr2)
	p.MapWriteNotices(func(wn *WriteNoticeRecord) { wn.Id = wn.Id + 1 }, 1, byte(0))
	assert.Equal(t, 2, wnr1.Id)
	assert.Equal(t, 3, wnr2.Id)
}

func TestPageArray1_SetDiff(t *testing.T) {
	b := byte(1)
	p := NewPageArray1(&b, 2)
	vc1 := NewVectorclock(2)
	vc2 := NewVectorclock(2)
	vc2.Increment(byte(1))
	int1 := &IntervalRecord{Timestamp: *vc1}
	wnr1 := &WriteNoticeRecord{Id: 1, Interval: int1}
	int2 := &IntervalRecord{Timestamp: *vc2}
	wnr2 := &WriteNoticeRecord{Id: 2, Interval: int2}
	p.AddWriteNoticeRecord(byte(1), 1, wnr1)
	p.AddWriteNoticeRecord(byte(1), 1, wnr2)

	diff1 := DiffDescription{
		ProcId:    byte(1),
		Timestamp: *vc1,
		Diff:      Diff{[]Pair{{byte(1), byte(2)}}},
	}
	diff2 := DiffDescription{
		ProcId:    byte(1),
		Timestamp: *vc2,
		Diff:      Diff{[]Pair{{byte(3), byte(4)}}},
	}
	//diffs := []DiffDescription{diff2, diff1}

	p.SetDiff(1, diff1)
	assert.Equal(t, &diff1.Diff, wnr1.Diff)
	p.SetDiff(1, diff2)
	assert.Equal(t, &diff1.Diff, wnr1.Diff)
	assert.Equal(t, &diff2.Diff, wnr2.Diff)
	wnr1.Diff = nil
	wnr2.Diff = nil
	assert.Equal(t, &diff1.Diff, wnr1.Diff)
	assert.Equal(t, &diff2.Diff, wnr2.Diff)
}

func TestPageArray1_GetWritenoticeRecords(t *testing.T) {
	b := byte(1)
	p := NewPageArray1(&b, 2)
	wnr1 := &WriteNoticeRecord{Id: 1}
	wnr2 := &WriteNoticeRecord{Id: 2}
	wnr3 := &WriteNoticeRecord{Id: 3}
	wnr4 := &WriteNoticeRecord{Id: 3}
	wnr5 := &WriteNoticeRecord{Id: 3}
	p.AddWriteNoticeRecord(byte(1), 1, wnr1)
	p.AddWriteNoticeRecord(byte(1), 1, wnr2)
	p.AddWriteNoticeRecord(byte(2), 1, wnr3)
	p.AddWriteNoticeRecord(byte(1), 0, wnr4)
	p.AddWriteNoticeRecord(byte(2), 0, wnr5)
	p.AddWriteNoticeRecord(byte(2), 3, wnr1)
	p.AddWriteNoticeRecord(byte(2), 3, wnr2)
	p.AddWriteNoticeRecord(byte(2), 3, wnr3)
	p.AddWriteNoticeRecord(byte(2), 3, wnr4)
	p.AddWriteNoticeRecord(byte(2), 3, wnr5)

	assert.Equal(t, []*WriteNoticeRecord{wnr2, wnr1}, p.GetWritenoticeRecords(byte(1), 1))
	assert.Equal(t, []*WriteNoticeRecord{wnr3}, p.GetWritenoticeRecords(byte(2), 1))
	assert.Equal(t, []*WriteNoticeRecord{wnr4}, p.GetWritenoticeRecords(byte(1), 0))
	assert.Equal(t, []*WriteNoticeRecord{wnr5}, p.GetWritenoticeRecords(byte(2), 0))
	assert.Equal(t, []*WriteNoticeRecord{wnr5, wnr4, wnr3, wnr2, wnr1},
		p.GetWritenoticeRecords(byte(2), 3))
}

/*
func TestDiffIterator_Insert(t *testing.T) {

	// For this testcase, the vc have the following order:
	// vc1 < vc2 < vc3
	// vc3 ~ vc4
	vc1 := NewVectorclock(3)
	vc2 := NewVectorclock(3)
	vc2.SetTick(byte(1), 1)
	vc3 := NewVectorclock(3)
	vc3.SetTick(byte(1), 2)
	vc4 := NewVectorclock(3)
	vc4.SetTick(byte(2), 2)

	smallProc := byte(1)
	mediumProc := byte(2)
	largeProc := byte(3)
	smallInt := &IntervalRecord{Timestamp: *vc1}
	mediumInt := &IntervalRecord{Timestamp: *vc2}
	largeInt := &IntervalRecord{Timestamp: *vc3}
	conInt1 := &IntervalRecord{Timestamp: *vc3}
	conInt2 := &IntervalRecord{Timestamp: *vc4}
	smallWrite := &WriteNoticeRecord{Id: 1, Interval: smallInt, Diff: &Diff{[]Pair{{byte(1), byte(1)}}}}
	mediumWrite := &WriteNoticeRecord{Id: 2, Interval: mediumInt, Diff: &Diff{[]Pair{{byte(2), byte(2)}}}}
	largeWrite := &WriteNoticeRecord{Id: 3, Interval: largeInt, Diff: &Diff{[]Pair{{byte(3), byte(3)}}}}
	/*conWrite1 := &WriteNoticeRecord{Id: 4, Interval: conInt1, Diff: &Diff{[]Pair{{byte(3), byte(3)}}}}
	conWrite2 := &WriteNoticeRecord{Id: 5, Interval: conInt2, Diff: &Diff{[]Pair{{byte(4), byte(4)}}}}

	//First all tests will have diffs.
	b := byte(1)
	pe := NewPageArrayEntry1(&b, 3, 1)
	pe.AddWritenoticeRecord(smallProc, smallWrite)
	pe.AddWritenoticeRecord(mediumProc, mediumWrite)
	pe.AddWritenoticeRecord(largeProc, largeWrite)
	/*di := pe.NewDiffIterator()
	di.Insert(smallProc)
	di.Insert(mediumProc)
	di.Insert(largeProc)
	assert.Equal(t, []byte{smallProc, mediumProc, largeProc}, di.order)
	assert.Equal(t, smallWrite.Diff, di.Next())
	assert.Equal(t, mediumWrite.Diff, di.Next())
	assert.Equal(t, largeWrite.Diff, di.Next())
	assert.Nil(t, di.Next())
	di = pe.NewDiffIterator()
	di.Insert(smallProc)
	di.Insert(largeProc)
	di.Insert(mediumProc)
	assert.Equal(t, []byte{smallProc, mediumProc, largeProc}, di.order)
	assert.Equal(t, smallWrite.Diff, di.Next())
	assert.Equal(t, mediumWrite.Diff, di.Next())
	assert.Equal(t, largeWrite.Diff, di.Next())
	assert.Nil(t, di.Next())
	di = pe.NewDiffIterator()
	di.Insert(mediumProc)
	di.Insert(smallProc)
	di.Insert(largeProc)
	assert.Equal(t, []byte{smallProc, mediumProc, largeProc}, di.order)
	assert.Equal(t, smallWrite.Diff, di.Next())
	assert.Equal(t, mediumWrite.Diff, di.Next())
	assert.Equal(t, largeWrite.Diff, di.Next())
	assert.Nil(t, di.Next())
	di = pe.NewDiffIterator()
	di.Insert(mediumProc)
	di.Insert(largeProc)
	di.Insert(smallProc)
	assert.Equal(t, []byte{smallProc, mediumProc, largeProc}, di.order)
	assert.Equal(t, smallWrite.Diff, di.Next())
	assert.Equal(t, mediumWrite.Diff, di.Next())
	assert.Equal(t, largeWrite.Diff, di.Next())
	assert.Nil(t, di.Next())
	di = pe.NewDiffIterator()
	di.Insert(largeProc)
	di.Insert(smallProc)
	di.Insert(mediumProc)
	assert.Equal(t, []byte{smallProc, mediumProc, largeProc}, di.order)
	assert.Equal(t, smallWrite.Diff, di.Next())
	assert.Equal(t, mediumWrite.Diff, di.Next())
	assert.Equal(t, largeWrite.Diff, di.Next())
	assert.Nil(t, di.Next())
	di = pe.NewDiffIterator()
	di.Insert(largeProc)
	di.Insert(mediumProc)
	di.Insert(smallProc)
	assert.Equal(t, []byte{smallProc, mediumProc, largeProc}, di.order)
	assert.Equal(t, smallWrite.Diff, di.Next())
	assert.Equal(t, mediumWrite.Diff, di.Next())
	assert.Equal(t, largeWrite.Diff, di.Next())
	assert.Nil(t, di.Next())

	pe = NewPageArrayEntry1(&b, 4, 1)
	pe.AddWritenoticeRecord(largeProc, smallWrite)
	pe.AddWritenoticeRecord(smallProc, mediumWrite)
	pe.AddWritenoticeRecord(mediumProc, largeWrite)
	di = pe.NewDiffIterator()
	di.Insert(smallProc)
	di.Insert(mediumProc)
	di.Insert(largeProc)
	assert.Equal(t, []byte{largeProc, smallProc, mediumProc}, di.order)
	assert.Equal(t, smallWrite.Diff, di.Next())
	assert.Equal(t, mediumWrite.Diff, di.Next())
	assert.Equal(t, largeWrite.Diff, di.Next())
	assert.Nil(t, di.Next())

	pe = NewPageArrayEntry1(&b, 4, 1)
	pe.AddWritenoticeRecord(smallProc, conWrite1)
	pe.AddWritenoticeRecord(largeProc, conWrite2)
	di = pe.NewDiffIterator()
	di.Insert(smallProc)
	di.Insert(largeProc)
	assert.Equal(t, []byte{smallProc, largeProc}, di.order)
	assert.Equal(t, conWrite1.Diff, di.Next())
	assert.Equal(t, conWrite2.Diff, di.Next())
	assert.Nil(t, di.Next())
	di = pe.NewDiffIterator()
	di.Insert(largeProc)
	di.Insert(smallProc)
	assert.Equal(t, []byte{smallProc, largeProc}, di.order)
	assert.Equal(t, conWrite1.Diff, di.Next())
	assert.Equal(t, conWrite2.Diff, di.Next())
	assert.Nil(t, di.Next())

}
func TestPageArrayEntry1_GetMissingDiffTimestamps(t *testing.T) {
	b := byte(1)
	pe := NewPageArrayEntry1(&b, 4, 1)
	nilPairs := []Pair{
		{nil, nil},
		{nil, nil},
		{nil, nil},
		{nil, nil},
	}
	assert.Equal(t, nilPairs, pe.GetMissingDiffTimestamps())

	// The vectorclocks follow the following rules:
	// vc1 < vc2 < vc3 < vc4

	vc0 := NewVectorclock(4)
	vc0.SetTick(byte(1), 1)
	vc1 := NewVectorclock(4)
	vc1.SetTick(byte(0), 1)
	vc2 := NewVectorclock(4)
	vc2.SetTick(byte(0), 2)
	vc3 := NewVectorclock(4)
	vc3.SetTick(byte(0), 2)
	vc3.SetTick(byte(1), 3)
	vc4 := NewVectorclock(4)
	vc4.SetTick(byte(0), 2)
	vc4.SetTick(byte(1), 4)

	vc5 := NewVectorclock(4)
	vc5.SetTick(byte(0), 2)
	vc5.SetTick(byte(1), 4)
	vc5.SetTick(byte(2), 1)

	vc6 := NewVectorclock(4)
	vc6.SetTick(byte(0), 2)
	vc6.SetTick(byte(1), 4)
	vc6.SetTick(byte(3), 1)

	proc1 := byte(0)
	proc2 := byte(1)

	int1 := &IntervalRecord{Timestamp: *vc1}
	int2 := &IntervalRecord{Timestamp: *vc2}
	int3 := &IntervalRecord{Timestamp: *vc3}
	int4 := &IntervalRecord{Timestamp: *vc4}
	int5 := &IntervalRecord{Timestamp: *vc5}
	int6 := &IntervalRecord{Timestamp: *vc6}

	write1 := &WriteNoticeRecord{Id: 1, Interval: int1}
	write2 := &WriteNoticeRecord{Id: 2, Interval: int2}
	write3 := &WriteNoticeRecord{Id: 3, Interval: int3}
	write4 := &WriteNoticeRecord{Id: 4, Interval: int4}
	write5 := &WriteNoticeRecord{Id: 5, Interval: int5}
	write6 := &WriteNoticeRecord{Id: 6, Interval: int6}

	//First we try all setups of concurrent stuff.
	pe = NewPageArrayEntry1(&b, 4, 1)
	pe.AddWritenoticeRecord(proc1, write1)
	pe.AddWritenoticeRecord(proc1, write2)
	pe.AddWritenoticeRecord(proc2, write3)
	pe.AddWritenoticeRecord(proc2, write4)
	expected := []Pair{
		{nil, nil},
		{*vc4, *vc1},
		{nil, nil},
		{nil, nil},
	}
	assert.Equal(t, expected, pe.GetMissingDiffTimestamps())

	pe = NewPageArrayEntry1(&b, 4, 1)
	pe.AddWritenoticeRecord(proc1, write1)
	pe.AddWritenoticeRecord(proc2, write2)
	pe.AddWritenoticeRecord(proc1, write3)
	pe.AddWritenoticeRecord(proc2, write4)
	expected = []Pair{
		{nil, nil},
		{*vc4, *vc1},
		{nil, nil},
		{nil, nil},
	}
	assert.Equal(t, expected, pe.GetMissingDiffTimestamps())

	pe = NewPageArrayEntry1(&b, 4, 1)
	pe.AddWritenoticeRecord(proc1, write1)
	pe.AddWritenoticeRecord(proc2, write2)
	pe.AddWritenoticeRecord(proc2, write3)
	pe.AddWritenoticeRecord(proc1, write4)
	expected = []Pair{
		{*vc4, *vc1},
		{nil, nil},
		{nil, nil},
		{nil, nil},
	}
	assert.Equal(t, expected, pe.GetMissingDiffTimestamps())

	pe = NewPageArrayEntry1(&b, 4, 1)
	pe.AddWritenoticeRecord(proc1, write1)
	pe.AddWritenoticeRecord(proc2, write2)
	pe.AddWritenoticeRecord(proc2, write3)
	pe.AddWritenoticeRecord(proc1, write4)
	expected = []Pair{
		{*vc4, *vc1},
		{nil, nil},
		{nil, nil},
		{nil, nil},
	}
	assert.Equal(t, expected, pe.GetMissingDiffTimestamps())

	pe = NewPageArrayEntry1(&b, 4, 1)

	pe.AddWritenoticeRecord(proc1, write1)
	pe.AddWritenoticeRecord(proc1, write2)
	pe.AddWritenoticeRecord(proc2, write3)
	pe.AddWritenoticeRecord(proc2, write4)
	pe.AddWritenoticeRecord(proc1, write5)
	pe.AddWritenoticeRecord(proc2, write6)

	expected = []Pair{
		{*vc5, *vc1},
		{*vc6, *vc3},
		{nil, nil},
		{nil, nil},
	}
	assert.Equal(t, expected, pe.GetMissingDiffTimestamps())

	pe = NewPageArrayEntry1(&b, 4, 1)
	write1.Diff = &Diff{}
	pe.AddWritenoticeRecord(proc1, write1)
	pe.AddWritenoticeRecord(proc1, write2)
	pe.AddWritenoticeRecord(proc2, write3)
	pe.AddWritenoticeRecord(proc2, write4)
	pe.AddWritenoticeRecord(proc1, write5)
	pe.AddWritenoticeRecord(proc2, write6)

	expected = []Pair{
		{*vc5, *vc2},
		{*vc6, *vc3},
		{nil, nil},
		{nil, nil},
	}
	assert.Equal(t, expected, pe.GetMissingDiffTimestamps())

	pe = NewPageArrayEntry1(&b, 4, 1)
	write1.Diff = &Diff{}
	write2.Diff = &Diff{}
	pe.AddWritenoticeRecord(proc1, write1)
	pe.AddWritenoticeRecord(proc1, write2)
	pe.AddWritenoticeRecord(proc2, write3)
	pe.AddWritenoticeRecord(proc2, write4)
	pe.AddWritenoticeRecord(proc1, write5)
	pe.AddWritenoticeRecord(proc2, write6)

	expected = []Pair{
		{*vc5, *vc5},
		{*vc6, *vc3},
		{nil, nil},
		{nil, nil},
	}
	assert.Equal(t, expected, pe.GetMissingDiffTimestamps())

	pe = NewPageArrayEntry1(&b, 4, 1)
	write1.Diff = &Diff{}
	write2.Diff = &Diff{}
	write3.Diff = &Diff{}
	pe.AddWritenoticeRecord(proc1, write1)
	pe.AddWritenoticeRecord(proc1, write2)
	pe.AddWritenoticeRecord(proc2, write3)
	pe.AddWritenoticeRecord(proc2, write4)
	pe.AddWritenoticeRecord(proc1, write5)
	pe.AddWritenoticeRecord(proc2, write6)

	expected = []Pair{
		{*vc5, *vc5},
		{*vc6, *vc4},
		{nil, nil},
		{nil, nil},
	}
	assert.Equal(t, expected, pe.GetMissingDiffTimestamps())
}
*/
