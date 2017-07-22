package treadmarks

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPageArray(t *testing.T) {
	//vm := memory.NewVmem(1024, 64)
	pageArray := NewPageArray(5)
	wn1 := &WriteNoticeRecord{Diff: nil, Interval: nil}
	wn2 := &WriteNoticeRecord{Diff: nil, Interval: nil}
	pe := NewPageArrayEntry(5)
	pe.copySet = []int{1, 2}
	pageArray.SetPageEntry(1, pe)

	pe.writeNoticeRecordArray[0] = []*WriteNoticeRecord{wn1, wn2}

	assert.Equal(t, wn1, pageArray.GetWritenoticeList(0, 1)[0])
	assert.Equal(t, wn2, pageArray.GetWritenoticeList(0, 1)[1])
}

func TestPageArrayEntry_AddWriteNotice(t *testing.T) {
	pe := NewPageArrayEntry(1)
	wn1 := pe.PrependWriteNotice(byte(0), WriteNotice{})
	wn1.Id = 1
	assert.True(t, wn1 == pe.GetWriteNotice(0, 0))
	wn2 := pe.PrependWriteNotice(byte(0), WriteNotice{})
	wn2.Id = 2
	assert.True(t, wn2 == pe.GetWriteNotice(0, 0))
	assert.Equal(t, wn1, pe.GetWriteNotice(0, 1))
}

func TestPageArrayEntry_DiffOrderChannel_TwoSameProc(t *testing.T) {
	pe := NewPageArrayEntry(3)

	//First we make three vectorclocks.
	vc1 := NewVectorclock(3)
	vc1.SetTick(byte(0), 1)
	vc2 := NewVectorclock(3)
	vc2.SetTick(byte(0), 2)

	//Then we make the interval record
	ir0 := &IntervalRecord{Timestamp: *vc1, WriteNotices: make([]*WriteNoticeRecord, 0)}
	ir1 := &IntervalRecord{Timestamp: *vc2, WriteNotices: make([]*WriteNoticeRecord, 0)}

	Diff1 := &Diff{[]Pair{{byte(1), byte(1)}}}
	Diff2 := &Diff{[]Pair{{byte(2), byte(2)}}}

	wr1 := &WriteNoticeRecord{
		Id:          1,
		WriteNotice: WriteNotice{1},
		Interval:    ir0,
		Diff:        Diff1,
	}
	wr2 := &WriteNoticeRecord{
		Id:          1,
		WriteNotice: WriteNotice{1},
		Interval:    ir1,
		Diff:        Diff2,
	}

	pe.writeNoticeRecordArray[0] = []*WriteNoticeRecord{wr2, wr1}

	channel := pe.OrderedDiffChannel()
	assert.Equal(t, Diff1, <-channel)
	assert.Equal(t, Diff2, <-channel)
}

func TestPageArrayEntry_DiffOrderChannel_TwoDifferentProc(t *testing.T) {
	pe := NewPageArrayEntry(3)

	//First we make three vectorclocks.
	vc1 := NewVectorclock(3)
	vc1.SetTick(byte(0), 1)
	vc2 := NewVectorclock(3)
	vc2.SetTick(byte(0), 2)

	//Then we make the interval record
	ir0 := &IntervalRecord{Timestamp: *vc1, WriteNotices: make([]*WriteNoticeRecord, 0)}
	ir1 := &IntervalRecord{Timestamp: *vc2, WriteNotices: make([]*WriteNoticeRecord, 0)}

	Diff1 := &Diff{[]Pair{{byte(1), byte(1)}}}
	Diff2 := &Diff{[]Pair{{byte(2), byte(2)}}}

	wr1 := &WriteNoticeRecord{
		Id:          1,
		WriteNotice: WriteNotice{1},
		Interval:    ir0,
		Diff:        Diff1,
	}
	wr2 := &WriteNoticeRecord{
		Id:          1,
		WriteNotice: WriteNotice{1},
		Interval:    ir1,
		Diff:        Diff2,
	}

	pe.writeNoticeRecordArray[0] = []*WriteNoticeRecord{wr1}
	pe.writeNoticeRecordArray[1] = []*WriteNoticeRecord{wr2}

	channel := pe.OrderedDiffChannel()
	assert.Equal(t, Diff1, <-channel)
	assert.Equal(t, Diff2, <-channel)
}

func TestPageArrayEntry_DiffOrderChannel_TwoDifferentProcReverse(t *testing.T) {
	pe := NewPageArrayEntry(3)

	//First we make three vectorclocks.
	vc1 := NewVectorclock(3)
	vc1.SetTick(byte(0), 1)
	vc2 := NewVectorclock(3)
	vc2.SetTick(byte(0), 2)

	//Then we make the interval record
	ir0 := &IntervalRecord{Timestamp: *vc1, WriteNotices: make([]*WriteNoticeRecord, 0)}
	ir1 := &IntervalRecord{Timestamp: *vc2, WriteNotices: make([]*WriteNoticeRecord, 0)}

	Diff1 := &Diff{[]Pair{{byte(1), byte(1)}}}
	Diff2 := &Diff{[]Pair{{byte(2), byte(2)}}}

	wr1 := &WriteNoticeRecord{
		Id:          1,
		WriteNotice: WriteNotice{1},
		Interval:    ir0,
		Diff:        Diff1,
	}
	wr2 := &WriteNoticeRecord{
		Id:          1,
		WriteNotice: WriteNotice{1},
		Interval:    ir1,
		Diff:        Diff2,
	}

	pe.writeNoticeRecordArray[1] = []*WriteNoticeRecord{wr1}
	pe.writeNoticeRecordArray[0] = []*WriteNoticeRecord{wr2}

	channel := pe.OrderedDiffChannel()
	assert.Equal(t, Diff1, <-channel)
	assert.Equal(t, Diff2, <-channel)
}

func TestPageArrayEntry_DiffOrderChannel_TwoDifferentProcConcurrent(t *testing.T) {
	pe := NewPageArrayEntry(3)

	//First we make three vectorclocks.
	vc1 := NewVectorclock(3)
	vc1.SetTick(byte(0), 1)
	vc2 := NewVectorclock(3)
	vc2.SetTick(byte(1), 2)

	//Then we make the interval record
	ir0 := &IntervalRecord{Timestamp: *vc1, WriteNotices: make([]*WriteNoticeRecord, 0)}
	ir1 := &IntervalRecord{Timestamp: *vc2, WriteNotices: make([]*WriteNoticeRecord, 0)}

	Diff1 := &Diff{[]Pair{{byte(1), byte(1)}}}
	Diff2 := &Diff{[]Pair{{byte(2), byte(2)}}}

	wr1 := &WriteNoticeRecord{
		Id:          1,
		WriteNotice: WriteNotice{1},
		Interval:    ir0,
		Diff:        Diff1,
	}
	wr2 := &WriteNoticeRecord{
		Id:          1,
		WriteNotice: WriteNotice{1},
		Interval:    ir1,
		Diff:        Diff2,
	}

	pe.writeNoticeRecordArray[0] = []*WriteNoticeRecord{wr1}
	pe.writeNoticeRecordArray[1] = []*WriteNoticeRecord{wr2}

	channel := pe.OrderedDiffChannel()
	assert.Equal(t, Diff1, <-channel)
	assert.Equal(t, Diff2, <-channel)
}

func TestPageArrayEntry_DiffOrderChannel_TwoDifferentProcReverseConcurrent(t *testing.T) {
	pe := NewPageArrayEntry(3)

	//First we make three vectorclocks.
	vc1 := NewVectorclock(3)
	vc1.SetTick(byte(0), 1)
	vc2 := NewVectorclock(3)
	vc2.SetTick(byte(1), 2)

	//Then we make the interval record
	ir0 := &IntervalRecord{Timestamp: *vc1, WriteNotices: make([]*WriteNoticeRecord, 0)}
	ir1 := &IntervalRecord{Timestamp: *vc2, WriteNotices: make([]*WriteNoticeRecord, 0)}

	Diff1 := &Diff{[]Pair{{byte(1), byte(1)}}}
	Diff2 := &Diff{[]Pair{{byte(2), byte(2)}}}

	wr1 := &WriteNoticeRecord{
		Id:          1,
		WriteNotice: WriteNotice{1},
		Interval:    ir0,
		Diff:        Diff1,
	}
	wr2 := &WriteNoticeRecord{
		Id:          1,
		WriteNotice: WriteNotice{1},
		Interval:    ir1,
		Diff:        Diff2,
	}

	pe.writeNoticeRecordArray[0] = []*WriteNoticeRecord{wr2}
	pe.writeNoticeRecordArray[1] = []*WriteNoticeRecord{wr1}

	channel := pe.OrderedDiffChannel()
	assert.Equal(t, Diff2, <-channel)
	assert.Equal(t, Diff1, <-channel)
}

/*
// This was just a proof of concept, to make sure pointers to writenoticerecords wouldnt break when
// the slice is extended.
func TestPageArrayEntry(t *testing.T) {
	s := []int{}
	for i := 0; i <= 8; i++ {
		s = append(s, i)
	}
	n := &s[4]
	fmt.Println(*n)
	*n = 6
	fmt.Println(s[4])
	fmt.Println(*n)
	s[4] = 2
	fmt.Println(s[4])
	fmt.Println(*n)
	for i := 9; i <= 10000; i++ {
		s = append([]int{i}, s...)
	}
	fmt.Println(*n)
}
*/
