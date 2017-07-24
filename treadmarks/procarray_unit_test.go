package treadmarks

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProcArray1_AddIntervalRecord(t *testing.T) {
	pa := NewProcArray(2)
	vc1 := NewVectorclock(2)
	vc2 := NewVectorclock(2)
	vc2.Increment(byte(1))
	ir1 := &IntervalRecord{
		procId:    byte(1),
		Timestamp: *vc1,
	}
	ir2 := &IntervalRecord{
		procId:    byte(1),
		Timestamp: *vc2,
	}
	wns1 := []*WriteNoticeRecord{
		{
			Id:          1,
			WriteNotice: WriteNotice{PageNr: 1},
			pageNr:      1,
			Interval:    ir1,
		},
		{
			Id:          2,
			WriteNotice: WriteNotice{PageNr: 2},
			pageNr:      2,
			Interval:    ir1,
		},
	}
	wns2 := []*WriteNoticeRecord{
		{
			Id:          3,
			WriteNotice: WriteNotice{PageNr: 1},
			pageNr:      1,
			Interval:    ir2,
		},
		{
			Id:          4,
			WriteNotice: WriteNotice{PageNr: 2},
			pageNr:      2,
			Interval:    ir2,
		},
	}
	ir1.WriteNotices = wns1
	ir2.WriteNotices = wns2
	assert.Len(t, pa.array[byte(1)], 0)
	pa.AddIntervalRecord(byte(1), ir1)
	assert.Len(t, pa.array[byte(1)], 1)
	assert.Equal(t, pa.array[byte(1)][0], ir1)
	pa.AddIntervalRecord(byte(1), ir2)
	assert.Len(t, pa.array[byte(1)], 2)
	assert.Equal(t, pa.array[byte(1)][0], ir1)
	assert.Equal(t, pa.array[byte(1)][1], ir2)
}

func TestProcArray1_GetIntervalRecord(t *testing.T) {
	pa := NewProcArray(2)
	vc1 := NewVectorclock(2)
	vc2 := NewVectorclock(2)
	vc2.Increment(byte(1))
	vc3 := NewVectorclock(2)
	vc3.Increment(byte(0))
	ir1 := &IntervalRecord{
		procId:    byte(1),
		Timestamp: *vc1,
	}
	ir2 := &IntervalRecord{
		procId:    byte(1),
		Timestamp: *vc2,
	}
	wns1 := []*WriteNoticeRecord{
		{
			Id:          1,
			WriteNotice: WriteNotice{PageNr: 1},
			pageNr:      1,
			Interval:    ir1,
		},
		{
			Id:          2,
			WriteNotice: WriteNotice{PageNr: 2},
			pageNr:      2,
			Interval:    ir1,
		},
	}
	wns2 := []*WriteNoticeRecord{
		{
			Id:          3,
			WriteNotice: WriteNotice{PageNr: 1},
			pageNr:      1,
			Interval:    ir2,
		},
		{
			Id:          4,
			WriteNotice: WriteNotice{PageNr: 2},
			pageNr:      2,
			Interval:    ir2,
		},
	}
	ir1.WriteNotices = wns1
	ir2.WriteNotices = wns2
	pa.AddIntervalRecord(byte(1), ir1)
	pa.AddIntervalRecord(byte(1), ir2)
	assert.Equal(t, ir1, pa.GetIntervalRecord(byte(1), *vc1))
	assert.Equal(t, ir2, pa.GetIntervalRecord(byte(1), *vc2))
	assert.Nil(t, pa.GetIntervalRecord(byte(0), *vc1))
	assert.Nil(t, pa.GetIntervalRecord(byte(1), *vc3))
}

func TestProcArray1_GetUnseenIntervalsAtProc(t *testing.T) {
	pa := NewProcArray(3)
	vc0 := NewVectorclock(3)
	vc1 := NewVectorclock(3)
	vc1.SetTick(byte(1), 1)
	vc2 := NewVectorclock(3)
	vc2.SetTick(byte(1), 2)
	vc3 := NewVectorclock(3)
	vc3.SetTick(byte(1), 5)

	vca := NewVectorclock(3)
	vca.SetTick(byte(0), 2)
	vcb := NewVectorclock(3)
	vcb.SetTick(byte(0), 2)
	vcb.SetTick(byte(1), 1)

	vcq := NewVectorclock(3)
	vcq.SetTick(byte(2), 1)
	ir1 := &IntervalRecord{
		procId:    byte(1),
		Timestamp: *vc1,
	}
	ir2 := &IntervalRecord{
		procId:    byte(1),
		Timestamp: *vc2,
	}
	ir3 := &IntervalRecord{
		procId:    byte(2),
		Timestamp: *vcq,
	}
	wns1 := []*WriteNoticeRecord{
		{
			Id:          1,
			WriteNotice: WriteNotice{PageNr: 1},
			pageNr:      1,
			Interval:    ir1,
		},
		{
			Id:          2,
			WriteNotice: WriteNotice{PageNr: 2},
			pageNr:      2,
			Interval:    ir1,
		},
	}
	wns2 := []*WriteNoticeRecord{
		{
			Id:          3,
			WriteNotice: WriteNotice{PageNr: 1},
			pageNr:      1,
			Interval:    ir2,
		},
		{
			Id:          4,
			WriteNotice: WriteNotice{PageNr: 2},
			pageNr:      2,
			Interval:    ir2,
		},
	}
	wns3 := []*WriteNoticeRecord{
		{
			Id:          3,
			WriteNotice: WriteNotice{PageNr: 1},
			pageNr:      1,
			Interval:    ir3,
		},
		{
			Id:          4,
			WriteNotice: WriteNotice{PageNr: 2},
			pageNr:      2,
			Interval:    ir3,
		},
	}
	ir1.WriteNotices = wns1
	ir2.WriteNotices = wns2
	ir3.WriteNotices = wns3

	int1 := Interval{
		Proc: byte(1),
		Vt:   *vc1,
		WriteNotices: []WriteNotice{
			{PageNr: 1},
			{PageNr: 2},
		},
	}
	int2 := Interval{
		Proc: byte(1),
		Vt:   *vc2,
		WriteNotices: []WriteNotice{
			{PageNr: 1},
			{PageNr: 2},
		},
	}
	int3 := Interval{
		Proc: byte(2),
		Vt:   *vcq,
		WriteNotices: []WriteNotice{
			{PageNr: 1},
			{PageNr: 2},
		},
	}
	assert.Len(t, pa.GetUnseenIntervalsAtProc(byte(1), *vc0), 0)

	pa.AddIntervalRecord(byte(1), ir1)
	pa.AddIntervalRecord(byte(1), ir2)
	pa.AddIntervalRecord(byte(2), ir3)

	//This tests intervals after timestamp.
	assert.Contains(t, pa.GetUnseenIntervalsAtProc(byte(1), *vc0), int1)
	assert.Contains(t, pa.GetUnseenIntervalsAtProc(byte(1), *vc0), int2)
	assert.Len(t, pa.GetUnseenIntervalsAtProc(byte(1), *vc0), 2)

	//This test, when an interval has the same timestamp as our parameter (we have already seen it).
	assert.Contains(t, pa.GetUnseenIntervalsAtProc(byte(1), *vc1), int2)
	assert.Len(t, pa.GetUnseenIntervalsAtProc(byte(1), *vc1), 1)

	//This tests if no intervals are unseen.
	assert.Len(t, pa.GetUnseenIntervalsAtProc(byte(1), *vc3), 0)

	//This tests when input is a concurrent timestamp thats before all entries.
	assert.Contains(t, pa.GetUnseenIntervalsAtProc(byte(1), *vca), int1)
	assert.Contains(t, pa.GetUnseenIntervalsAtProc(byte(1), *vca), int2)
	assert.Len(t, pa.GetUnseenIntervalsAtProc(byte(1), *vca), 2)

	//This tests when input is a concurrent timestamp thats after one of the entries.
	assert.Contains(t, pa.GetUnseenIntervalsAtProc(byte(1), *vcb), int2)
	assert.Len(t, pa.GetUnseenIntervalsAtProc(byte(1), *vcb), 1)

	//This tests that we dont get intervals from other procs.
	assert.Contains(t, pa.GetUnseenIntervalsAtProc(byte(2), *vc0), int3)
	assert.Len(t, pa.GetUnseenIntervalsAtProc(byte(2), *vc0), 1)
}

func TestProcArray1_GetAllUnseenIntervals(t *testing.T) {
	pa := NewProcArray(3)
	vc0 := NewVectorclock(3)
	vc1 := NewVectorclock(3)
	vc1.SetTick(byte(1), 1)
	vc2 := NewVectorclock(3)
	vc2.SetTick(byte(1), 2)
	vc3 := NewVectorclock(3)
	vc3.SetTick(byte(1), 5)

	vcq := NewVectorclock(3)
	vcq.SetTick(byte(2), 1)

	vcz := NewVectorclock(3)
	vcz.SetTick(byte(1), 2)
	vcz.SetTick(byte(2), 2)

	ir1 := &IntervalRecord{
		procId:    byte(1),
		Timestamp: *vc1,
	}
	ir2 := &IntervalRecord{
		procId:    byte(1),
		Timestamp: *vc2,
	}
	ir3 := &IntervalRecord{
		procId:    byte(2),
		Timestamp: *vcq,
	}
	wns1 := []*WriteNoticeRecord{
		{
			Id:          1,
			WriteNotice: WriteNotice{PageNr: 1},
			pageNr:      1,
			Interval:    ir1,
		},
		{
			Id:          2,
			WriteNotice: WriteNotice{PageNr: 2},
			pageNr:      2,
			Interval:    ir1,
		},
	}
	wns2 := []*WriteNoticeRecord{
		{
			Id:          3,
			WriteNotice: WriteNotice{PageNr: 1},
			pageNr:      1,
			Interval:    ir2,
		},
		{
			Id:          4,
			WriteNotice: WriteNotice{PageNr: 2},
			pageNr:      2,
			Interval:    ir2,
		},
	}
	wns3 := []*WriteNoticeRecord{
		{
			Id:          3,
			WriteNotice: WriteNotice{PageNr: 1},
			pageNr:      1,
			Interval:    ir3,
		},
		{
			Id:          4,
			WriteNotice: WriteNotice{PageNr: 2},
			pageNr:      2,
			Interval:    ir3,
		},
	}
	ir1.WriteNotices = wns1
	ir2.WriteNotices = wns2
	ir3.WriteNotices = wns3

	int1 := Interval{
		Proc: byte(1),
		Vt:   *vc1,
		WriteNotices: []WriteNotice{
			{PageNr: 1},
			{PageNr: 2},
		},
	}
	int2 := Interval{
		Proc: byte(1),
		Vt:   *vc2,
		WriteNotices: []WriteNotice{
			{PageNr: 1},
			{PageNr: 2},
		},
	}
	int3 := Interval{
		Proc: byte(2),
		Vt:   *vcq,
		WriteNotices: []WriteNotice{
			{PageNr: 1},
			{PageNr: 2},
		},
	}

	pa.AddIntervalRecord(byte(1), ir1)
	pa.AddIntervalRecord(byte(1), ir2)
	pa.AddIntervalRecord(byte(2), ir3)

	assert.Contains(t, pa.GetAllUnseenIntervals(*vc0), int1)
	assert.Contains(t, pa.GetAllUnseenIntervals(*vc0), int2)
	assert.Contains(t, pa.GetAllUnseenIntervals(*vc0), int3)
	assert.Len(t, pa.GetAllUnseenIntervals(*vc0), 3)

	assert.Contains(t, pa.GetAllUnseenIntervals(*vc1), int2)
	assert.Contains(t, pa.GetAllUnseenIntervals(*vc1), int3)
	assert.Len(t, pa.GetAllUnseenIntervals(*vc1), 2)

	assert.Contains(t, pa.GetAllUnseenIntervals(*vc2), int3)
	assert.Len(t, pa.GetAllUnseenIntervals(*vc2), 1)

	assert.Contains(t, pa.GetAllUnseenIntervals(*vc3), int3)
	assert.Len(t, pa.GetAllUnseenIntervals(*vc2), 1)

	assert.Contains(t, pa.GetAllUnseenIntervals(*vcq), int1)
	assert.Contains(t, pa.GetAllUnseenIntervals(*vcq), int2)
	assert.Len(t, pa.GetAllUnseenIntervals(*vcq), 2)

	assert.Len(t, pa.GetAllUnseenIntervals(*vcz), 0)
}

func TestProcArray1_MapIntervalRecords(t *testing.T) {
	pa := NewProcArray(3)
	vc1 := NewVectorclock(3)
	vc1.SetTick(byte(1), 1)
	vc2 := NewVectorclock(3)
	vc2.SetTick(byte(1), 2)

	vcq := NewVectorclock(3)
	vcq.SetTick(byte(2), 1)

	ir1 := &IntervalRecord{
		procId:    byte(1),
		Timestamp: *vc1,
	}
	ir2 := &IntervalRecord{
		procId:    byte(1),
		Timestamp: *vc2,
	}
	ir3 := &IntervalRecord{
		procId:    byte(2),
		Timestamp: *vcq,
	}

	pa.AddIntervalRecord(byte(1), ir1)
	pa.AddIntervalRecord(byte(1), ir2)
	pa.AddIntervalRecord(byte(2), ir3)

	pa.MapIntervalRecords(byte(1), func(ir *IntervalRecord) { ir.procId = ir.procId + 2 })
	assert.Equal(t, byte(3), ir1.procId)
	assert.Equal(t, byte(3), ir2.procId)
	assert.Equal(t, byte(2), ir3.procId)
}
