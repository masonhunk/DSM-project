package treadmarks

import (
	"DSM-project/memory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateDiff(t *testing.T) {
	original := []byte{0, 1, 2, 3, 4}
	n := []byte{0, 1, 3, 5, 4}
	assert.Equal(t, []Pair{{2, byte(3)}, {3, byte(5)}}, CreateDiff(original, n).Diffs)
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

func TestTreadMarks_handleLockAcquireRequest(t *testing.T) {
	vm := memory.NewVmem(128, 8)
	tm := NewTreadMarks(vm, 2, 1, 1)
	tm.procId = 1
	vc1 := NewVectorclock(2)
	vc2 := NewVectorclock(2)
	vc3 := NewVectorclock(2)
	vc1.Increment(byte(0))
	//First we make one interval record with matching write notice records
	ir1 := &IntervalRecord{Timestamp: *vc1, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr1_1 := tm.pageArray.PrependWriteNotice(byte(0), WriteNotice{pageNr: 0})
	wr1_2 := tm.pageArray.PrependWriteNotice(byte(0), WriteNotice{pageNr: 3})
	ir1.WriteNotices = []*WriteNoticeRecord{wr1_1, wr1_2}
	wr1_1.Interval = ir1
	wr1_2.Interval = ir1
	tm.procArray[0].PrependIntervalRecord(ir1)

	//Then we make another interval record with matching write notice records.
	vc2.Increment(byte(0))
	vc2.Increment(byte(0))
	ir2 := &IntervalRecord{Timestamp: *vc2, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr2_1 := tm.pageArray.PrependWriteNotice(byte(0), WriteNotice{pageNr: 2})
	wr2_2 := tm.pageArray.PrependWriteNotice(byte(0), WriteNotice{pageNr: 3})
	ir2.WriteNotices = []*WriteNoticeRecord{wr2_1, wr2_2}
	wr2_1.Interval = ir2
	wr2_2.Interval = ir2
	tm.procArray[0].PrependIntervalRecord(ir2)

	//Now we see how the host responds when we request a lock.
	msg := new(TM_Message)
	msg.From = 0
	msg.To = 1
	msg.Type = LOCK_ACQUIRE_REQUEST
	msg.VC = *vc3

	response := tm.HandleLockAcquireRequest(msg)

	int1 := Interval{
		0,
		Vectorclock{[]uint{2, 0}},
		[]WriteNotice{
			{2},
			{3}},
	}
	int2 := Interval{
		0,
		Vectorclock{[]uint{1, 0}},
		[]WriteNotice{
			{0},
			{3}},
	}
	assert.Equal(t, int1, response.Intervals[0])
	assert.Equal(t, int2, response.Intervals[1])

	msg.VC = *vc1
	response = tm.HandleLockAcquireRequest(msg)
	assert.Equal(t, int1, response.Intervals[0],
		"We should only recieve intervals later than our timestamp.")
}

func TestTreadMarks_GenerateDiffRequest(t *testing.T){
	vm := memory.NewVmem(128, 8)
	tm := NewTreadMarks(vm, 4, 1, 1)
	tm.procId = 1
	vc1 := NewVectorclock(4)
	vc2 := NewVectorclock(4)
	vc3 := NewVectorclock(4)
	vc4 := NewVectorclock(4)
	vc1.Increment(byte(0))
	//First we make one interval record with matching write notice records
	ir1 := &IntervalRecord{Timestamp: *vc1, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr1_1 := tm.pageArray.PrependWriteNotice(byte(2), WriteNotice{pageNr: 0})
	wr1_2 := tm.pageArray.PrependWriteNotice(byte(0), WriteNotice{pageNr: 3})
	ir1.WriteNotices = []*WriteNoticeRecord{wr1_1, wr1_2}
	wr1_1.Interval = ir1
	wr1_2.Interval = ir1
	tm.procArray[1].PrependIntervalRecord(ir1)

	//Then we make another interval record with matching write notice records.
	vc2.SetTick(byte(2), 4)
	ir2 := &IntervalRecord{Timestamp: *vc2, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr2_1 := tm.pageArray.PrependWriteNotice(byte(0), WriteNotice{pageNr: 1})
	wr2_2 := tm.pageArray.PrependWriteNotice(byte(1), WriteNotice{pageNr: 3})
	ir2.WriteNotices = []*WriteNoticeRecord{wr2_1, wr2_2}
	wr2_1.Interval = ir2
	wr2_2.Interval = ir2
	tm.procArray[1].PrependIntervalRecord(ir2)

	//Then we make another interval record with matching write notice records.
	vc3.SetTick(byte(3), 4)
	ir3 := &IntervalRecord{Timestamp: *vc3, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr3_1 := tm.pageArray.PrependWriteNotice(byte(0), WriteNotice{pageNr: 2})
	wr3_2 := tm.pageArray.PrependWriteNotice(byte(2), WriteNotice{pageNr: 3})
	ir3.WriteNotices = []*WriteNoticeRecord{wr3_1, wr3_2}
	wr3_1.Interval = ir3
	wr3_2.Interval = ir3
	tm.procArray[1].PrependIntervalRecord(ir3)

	result := tm.GenerateDiffRequests(0)
	assert.Len(t, result, 1)
	assert.Contains(t, result, TM_Message{From: tm.procId, To: byte(2), Type:DIFF_REQUEST, VC:*vc1, PageNr:0})
	result = tm.GenerateDiffRequests(1)
	assert.Len(t, result, 1)
	assert.Contains(t, result, TM_Message{From: tm.procId, To: byte(0), Type:DIFF_REQUEST, VC:*vc2, PageNr:1})
	result = tm.GenerateDiffRequests(2)
	assert.Len(t, result, 1)
	assert.Contains(t, result, TM_Message{From: tm.procId, To: byte(0), Type:DIFF_REQUEST, VC:*vc3, PageNr:2})

	result = tm.GenerateDiffRequests(3)
	assert.Contains(t, result, TM_Message{From: tm.procId, To: byte(2), Type:DIFF_REQUEST, VC:*vc3, PageNr:3})
	assert.Contains(t, result, TM_Message{From: tm.procId, To: byte(1), Type:DIFF_REQUEST, VC:*vc2, PageNr:3})
	assert.Contains(t, result, TM_Message{From: tm.procId, To: byte(0), Type:DIFF_REQUEST, VC:*vc1, PageNr:3})
	assert.Len(t, result, 3)

	//Then we make another interval record with matching write notice records.
	vc4.SetTick(byte(3), 5)
	vc4.SetTick(byte(2), 5)
	vc4.SetTick(byte(0), 5)
	ir4 := &IntervalRecord{Timestamp: *vc4, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr4_1 := tm.pageArray.PrependWriteNotice(byte(0), WriteNotice{pageNr: 2})
	wr4_2 := tm.pageArray.PrependWriteNotice(byte(2), WriteNotice{pageNr: 3})
	ir3.WriteNotices = []*WriteNoticeRecord{wr4_1, wr4_2}
	wr4_1.Interval = ir4
	wr4_2.Interval = ir4
	tm.procArray[1].PrependIntervalRecord(ir4)
	result = tm.GenerateDiffRequests(3)
	assert.Contains(t, result, TM_Message{From: tm.procId, To: byte(2), Type:DIFF_REQUEST, VC:*vc3, PageNr:3})
	assert.Len(t, result, 1)

	wr3_2.Diff = new(Diff)

	result = tm.GenerateDiffRequests(3)
	assert.Contains(t, result, TM_Message{From: tm.procId, To: byte(2), Type:DIFF_REQUEST, VC:*vc4, PageNr:3})
	assert.Len(t, result, 1)
}
