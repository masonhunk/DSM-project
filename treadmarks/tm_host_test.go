package treadmarks

import (
	"DSM-project/memory"
	"DSM-project/network"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

//TODO remove when done. Is only there to make the compiler shut up.
var _ = fmt.Print

func TestCreateDiff(t *testing.T) {
	original := []byte{0, 1, 2, 3, 4}
	n := []byte{0, 1, 3, 5, 4}
	assert.Equal(t, []Pair{{2, byte(3)}, {3, byte(5)}}, CreateDiff(original, n).Diffs)
}

func TestUpdateDatastructures(t *testing.T) {
	vm := memory.NewVmem(128, 8)
	tm := NewTreadMarks(vm, 1, 1, 1)
	tm.ProcId = 3
	tm.twinMap[0] = []byte{4, 4, 4, 4, 4, 4, 4, 4}
	tm.twinMap[1] = []byte{1, 1, 1, 1, 1, 1, 1, 1}
	procArray := MakeProcArray(5)
	tm.TM_IDataStructures = &TM_DataStructures{ProcArray: procArray, PageArray: *NewPageArray(5)}
	tm.updateDatastructures()
	headWNRecord := tm.GetWriteNoticeListHead(0, 3)
	//headWNRecord := tm.pageArray[0].ProcArr[3]
	//headIntervalRecord := tm.procArray[3].car
	headIntervalRecord := tm.GetIntervalRecordHead(3)
	assert.Len(t, headIntervalRecord.WriteNotices, 2)
	assert.True(t, headWNRecord == headIntervalRecord.WriteNotices[0])
	fmt.Println(headIntervalRecord)
	fmt.Println(headWNRecord.Interval)
	assert.Equal(t, headIntervalRecord, headWNRecord.Interval)

	//headWNRecord1 := tm.pageArray[1].ProcArr[3]
	headWNRecord1 := tm.GetWriteNoticeListHead(1, 3)

	assert.True(t, headWNRecord1 == headIntervalRecord.WriteNotices[1])
}

/*
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
*/
func TestTreadMarks_handleLockAcquireRequest(t *testing.T) {
	vm := memory.NewVmem(128, 8)
	tm := NewTreadMarks(vm, 2, 1, 1)
	tm.ProcId = 1
	vc1 := NewVectorclock(2)
	vc2 := NewVectorclock(2)
	vc3 := NewVectorclock(2)
	vc1.Increment(byte(0))
	//First we make one interval record with matching write notice records
	ir1 := &IntervalRecord{Timestamp: *vc1, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr1_1 := tm.PrependWriteNotice(byte(0), WriteNotice{PageNr: 0})
	wr1_2 := tm.PrependWriteNotice(byte(0), WriteNotice{PageNr: 3})
	ir1.WriteNotices = []*WriteNoticeRecord{wr1_1, wr1_2}
	wr1_1.Interval = ir1
	wr1_2.Interval = ir1
	tm.PrependIntervalRecord(0, ir1)

	//Then we make another interval record with matching write notice records.
	vc2.Increment(byte(0))
	vc2.Increment(byte(0))
	ir2 := &IntervalRecord{Timestamp: *vc2, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr2_1 := tm.PrependWriteNotice(byte(0), WriteNotice{PageNr: 2})
	wr2_2 := tm.PrependWriteNotice(byte(0), WriteNotice{PageNr: 3})
	ir2.WriteNotices = []*WriteNoticeRecord{wr2_1, wr2_2}
	wr2_1.Interval = ir2
	wr2_2.Interval = ir2
	tm.PrependIntervalRecord(0, ir2)

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
	assert.Equal(t, int2, response.Intervals[1])
	assert.Equal(t, int1, response.Intervals[0])

	msg.VC = *vc1
	response = tm.HandleLockAcquireRequest(msg)
	assert.Equal(t, int1, response.Intervals[0],
		"We should only recieve intervals later than our timestamp.")
}

/*
func TestTreadMarks_GenerateDiffRequest(t *testing.T) {
	vm := memory.NewVmem(128, 8)
	tm := NewTreadMarks(vm, 4, 1, 1)
	tm.ProcId = 1
	vc1 := NewVectorclock(4)
	vc2 := NewVectorclock(4)
	vc3 := NewVectorclock(4)
	vc4 := NewVectorclock(4)
	vc1.Increment(byte(0))
	//First we make one interval record with matching write notice records
	ir1 := &IntervalRecord{Timestamp: *vc1, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr1_1 := tm.TM_IDataStructures.PrependWriteNotice(byte(2), WriteNotice{pageNr: 0})
	wr1_2 := tm.TM_IDataStructures.PrependWriteNotice(byte(0), WriteNotice{pageNr: 3})
	ir1.WriteNotices = []*WriteNoticeRecord{wr1_1, wr1_2}
	wr1_1.Interval = ir1
	wr1_2.Interval = ir1
	tm.TM_IDataStructures.PrependIntervalRecord(byte(1), ir1)

	//Then we make another interval record with matching write notice records.
	vc2.SetTick(byte(2), 4)
	ir2 := &IntervalRecord{Timestamp: *vc2, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr2_1 := tm.TM_IDataStructures.PrependWriteNotice(byte(0), WriteNotice{pageNr: 1})
	wr2_2 := tm.TM_IDataStructures.PrependWriteNotice(byte(1), WriteNotice{pageNr: 3})
	ir2.WriteNotices = []*WriteNoticeRecord{wr2_1, wr2_2}
	wr2_1.Interval = ir2
	wr2_2.Interval = ir2
	tm.TM_IDataStructures.PrependIntervalRecord(byte(1), ir2)

	//Then we make another interval record with matching write notice records.
	vc3.SetTick(byte(3), 4)
	ir3 := &IntervalRecord{Timestamp: *vc3, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr3_1 := tm.TM_IDataStructures.PrependWriteNotice(byte(0), WriteNotice{pageNr: 2})
	wr3_2 := tm.TM_IDataStructures.PrependWriteNotice(byte(2), WriteNotice{pageNr: 3})
	ir3.WriteNotices = []*WriteNoticeRecord{wr3_1, wr3_2}
	wr3_1.Interval = ir3
	wr3_2.Interval = ir3
	tm.TM_IDataStructures.PrependIntervalRecord(byte(1), ir3)

	result := tm.GenerateDiffRequests(0)
	assert.Len(t, result, 1)
	assert.Contains(t, result, TM_Message{From: tm.ProcId, To: byte(2), Type: DIFF_REQUEST, VC: *vc1, PageNr: 0})
	result = tm.GenerateDiffRequests(1)
	assert.Len(t, result, 1)
	assert.Contains(t, result, TM_Message{From: tm.ProcId, To: byte(0), Type: DIFF_REQUEST, VC: *vc2, PageNr: 1})
	result = tm.GenerateDiffRequests(2)
	assert.Len(t, result, 1)
	assert.Contains(t, result, TM_Message{From: tm.ProcId, To: byte(0), Type: DIFF_REQUEST, VC: *vc3, PageNr: 2})

	result = tm.GenerateDiffRequests(3)
	assert.Contains(t, result, TM_Message{From: tm.ProcId, To: byte(2), Type: DIFF_REQUEST, VC: *vc3, PageNr: 3})
	assert.Contains(t, result, TM_Message{From: tm.ProcId, To: byte(1), Type: DIFF_REQUEST, VC: *vc2, PageNr: 3})
	assert.Contains(t, result, TM_Message{From: tm.ProcId, To: byte(0), Type: DIFF_REQUEST, VC: *vc1, PageNr: 3})
	assert.Len(t, result, 3)

	//Then we make another interval record with matching write notice records.
	vc4.SetTick(byte(3), 5)
	vc4.SetTick(byte(2), 5)
	vc4.SetTick(byte(0), 5)
	ir4 := &IntervalRecord{Timestamp: *vc4, WriteNotices: make([]*WriteNoticeRecord, 0)}
	wr4_1 := tm.TM_IDataStructures.PrependWriteNotice(byte(0), WriteNotice{pageNr: 2})
	wr4_2 := tm.TM_IDataStructures.PrependWriteNotice(byte(2), WriteNotice{pageNr: 3})
	ir3.WriteNotices = []*WriteNoticeRecord{wr4_1, wr4_2}
	wr4_1.Interval = ir4
	wr4_2.Interval = ir4
	tm.TM_IDataStructures.PrependIntervalRecord(byte(1), ir4)
	result = tm.GenerateDiffRequests(3)
	assert.Contains(t, result, TM_Message{From: tm.ProcId, To: byte(2), Type: DIFF_REQUEST, VC: *vc3, PageNr: 3})
	assert.Len(t, result, 1)

	wr3_2.Diff = new(Diff)

	result = tm.GenerateDiffRequests(3)
	assert.Contains(t, result, TM_Message{From: tm.ProcId, To: byte(2), Type: DIFF_REQUEST, VC: *vc4, PageNr: 3})
	assert.Len(t, result, 1)
}
*/
func TestApplyingIntervalsToDataStructure(t *testing.T) {
	tm := NewTreadMarks(memory.NewVmem(128, 8), 4, 1, 1)
	tm.ProcId = byte(2) //this host id
	msg := TM_Message{
		//test non-overlapping intervals
		//newest interval first when from same process
		Intervals: []Interval{
			{
				Vt:           Vectorclock{[]uint{1, 0, 0, 0}},
				Proc:         byte(0),
				WriteNotices: []WriteNotice{{1}, {2}, {3}},
			},
			{
				Vt:           Vectorclock{[]uint{1, 2, 0, 0}},
				Proc:         byte(1),
				WriteNotices: []WriteNotice{{1}, {3}, {4}},
			},
			{
				Vt:           Vectorclock{[]uint{0, 1, 0, 0}},
				Proc:         byte(1),
				WriteNotices: []WriteNotice{{1}, {5}, {6}},
			},
		},
	}
	tm.incorporateIntervalsIntoDatastructures(&msg)
	assert.Equal(t, Vectorclock{[]uint{1, 0, 0, 0}}, tm.GetIntervalRecordHead(0).Timestamp)
	assert.Equal(t, Vectorclock{[]uint{1, 2, 0, 0}}, tm.GetIntervalRecordHead(1).Timestamp)
	assert.Equal(t, Vectorclock{[]uint{0, 1, 0, 0}}, tm.GetIntervalRecord(byte(1), 1).Timestamp)

	assert.Equal(t, tm.GetIntervalRecordHead(0).WriteNotices[0], tm.GetWriteNoticeListHead(1, 0))
	assert.Equal(t, tm.GetIntervalRecordHead(0).WriteNotices[1], tm.GetWriteNoticeListHead(2, 0))
	assert.Equal(t, tm.GetIntervalRecordHead(0).WriteNotices[2], tm.GetWriteNoticeListHead(3, 0))

	assert.Equal(t, tm.GetIntervalRecordHead(1).WriteNotices[0], tm.GetWriteNoticeListHead(1, 1))
	assert.Equal(t, tm.GetIntervalRecordHead(1).WriteNotices[1], tm.GetWriteNoticeListHead(3, 1))

	//assert.Equal(t, *tm.GetIntervalRecord(1, 1).WriteNotices[0], tm.GetWritenotices(1, 1)[1])

}

func TestShouldRequestCopyIfNoCopy(t *testing.T) {
	tm := NewTreadMarks(memory.NewVmem(128, 8), 4, 1, 1)
	tm.ProcId = byte(2)
	cm := NewClientMock()
	tm.IClient = cm
	tm.Connect("")
	tm.Read(50)

	assert.Len(t, cm.messages, 1)
	assert.Equal(t, COPY_REQUEST, cm.messages[0].Type)
	assert.Equal(t, 50/8, cm.messages[0].PageNr)
}

func TestShouldSendCopyOnRequest(t *testing.T) {
	tm := NewTreadMarks(memory.NewVmem(128, 8), 4, 1, 1)
	tm.ProcId = byte(2)
	tm.Startup()
	cm := NewClientMock()
	tm.IClient = cm
	cm.handler = func(msg TM_Message) {
		if pg, ok := tm.twinMap[msg.PageNr]; ok {
			msg.Data = pg
		} else {
			tm.PrivilegedRead(msg.PageNr*tm.GetPageSize(), tm.GetPageSize())
			msg.Data = pg
		}
		msg.From, msg.To = msg.To, msg.From
		err := tm.Send(msg)
		panicOnErr(err)

	}
	tm.Connect("")
	cm.handler(TM_Message{Type: COPY_REQUEST, From: byte(1), To: byte(2), PageNr: 5})
	assert.Len(t, cm.messages, 1)

}

type ClientMock struct {
	messages []TM_Message
	handler  func(msg TM_Message)
}

func (c *ClientMock) Send(message network.Message) error {
	msg := message.(TM_Message)
	c.messages = append(c.messages, msg)
	go func() {
		time.Sleep(time.Millisecond * 100)
		*msg.Event <- "ok"
	}()
	return nil
}

func (c *ClientMock) GetTransciever() network.ITransciever {
	panic("implement me")
}

func (c *ClientMock) Connect(address string) error {
	return nil
}

func (c *ClientMock) Close() {
}

func NewClientMock() *ClientMock {
	cMock := new(ClientMock)
	cMock.messages = make([]TM_Message, 0)

	return cMock
}

func SetupHandleDiffRequest() *TreadMarks {
	vm := memory.NewVmem(128, 8)
	tm := NewTreadMarks(vm, 2, 1, 1)
	tm.ProcId = 1

	//Setup
	//First we make three vectorclocks.
	vc := NewVectorclock(3)
	vc.SetTick(byte(1), 3)
	tm.vc = *vc

	//Then we make the interval record
	ir0 := &IntervalRecord{Timestamp: *vc, WriteNotices: make([]*WriteNoticeRecord, 0)}
	ir1 := &IntervalRecord{Timestamp: *vc, WriteNotices: make([]*WriteNoticeRecord, 0)}

	//Then the writenoticerecords
	wr1 := tm.PrependWriteNotice(byte(0), WriteNotice{PageNr: 0})
	wr2 := tm.PrependWriteNotice(byte(0), WriteNotice{PageNr: 1})
	wr3 := tm.PrependWriteNotice(byte(1), WriteNotice{PageNr: 0})
	wr4 := tm.PrependWriteNotice(byte(1), WriteNotice{PageNr: 1})
	//We add the writenoticerecords to the interval record.
	ir0.WriteNotices = []*WriteNoticeRecord{wr1, wr2}
	ir1.WriteNotices = []*WriteNoticeRecord{wr3, wr4}

	//Then we fix the interval record pointer for each of the writenoticerecords
	wr1.Interval = ir0
	wr2.Interval = ir0
	wr3.Interval = ir1
	wr4.Interval = ir1

	//Lastly we add a diff to two of the write notices.
	wr1.Diff = &Diff{[]Pair{{byte(0), byte(1)}}}
	wr2.Diff = &Diff{[]Pair{{byte(0), byte(2)}}}

	//In the end we add the two interval records.
	tm.PrependIntervalRecord(byte(0), ir0)
	tm.PrependIntervalRecord(byte(1), ir1)
	return tm
}

func TestTreadMarks_HandleDiffRequest_DiffsAfterRequestVC(t *testing.T) {
	tm := SetupHandleDiffRequest()
	vc := tm.vc

	//First we test when the diffs have a vectorclock later than our vectorclock.
	testVC := NewVectorclock(3)
	request := TM_Message{From: byte(0), To: byte(1), Type: DIFF_REQUEST, VC: *testVC, PageNr: 0}
	diffs := tm.HandleDiffRequest(request)
	assert.Equal(t, diffs.Diffs, []Pair{{vc, tm.GetWriteNoticeListHead(0, 0).Diff.Diffs}},
		"We should recieve the diffs of WR1, with the vectorclock ", vc)
	request = TM_Message{From: byte(0), To: byte(1), Type: DIFF_REQUEST, VC: *testVC, PageNr: 1}
	diffs = tm.HandleDiffRequest(request)
	assert.Equal(t, diffs.Diffs, []Pair{{vc, tm.GetWriteNoticeListHead(1, 0).Diff.Diffs}},
		"We should recieve the diffs of WR2, with the vectorclock ", vc)
}

func TestTreadMarks_HandleDiffRequest_DiffsVCConcurrentToRequestVC(t *testing.T) {
	tm := SetupHandleDiffRequest()
	vc := tm.vc

	testVC := NewVectorclock(3)

	//Now we make the vectorclock concurrent with the vectorclock of the writenotices.
	//We should recieve the same output as before.
	testVC.SetTick(byte(0), 3)
	request := TM_Message{From: byte(0), To: byte(1), Type: DIFF_REQUEST, VC: *testVC, PageNr: 0}
	diffs := tm.HandleDiffRequest(request)
	assert.Equal(t, diffs.Diffs, []Pair{{vc, tm.GetWriteNoticeListHead(0, 0).Diff.Diffs}},
		"We should recieve the diffs of WR1, with the vectorclock ", vc)
	request = TM_Message{From: byte(0), To: byte(1), Type: DIFF_REQUEST, VC: *testVC, PageNr: 1}
	diffs = tm.HandleDiffRequest(request)
	assert.Equal(t, diffs.Diffs, []Pair{{vc, tm.GetWriteNoticeListHead(1, 0).Diff.Diffs}},
		"We should recieve the diffs of WR2, with the vectorclock ", vc)

}

func TestTreadMarks_HandleDiffRequest_DiffVCEqualToRequestVC(t *testing.T) {
	tm := SetupHandleDiffRequest()
	vc := tm.vc

	testVC := NewVectorclock(3)

	//Now we make a vector timestamp that is equal to the former vector timestamp.
	//This should also give us the diffs.
	testVC.SetTick(byte(0), 0)
	testVC.SetTick(byte(1), 3)
	request := TM_Message{From: byte(0), To: byte(1), Type: DIFF_REQUEST, VC: *testVC, PageNr: 0}
	diffs := tm.HandleDiffRequest(request)
	assert.Equal(t, diffs.Diffs, []Pair{{vc, tm.GetWriteNoticeListHead(0, 0).Diff.Diffs}},
		"We should recieve the diffs of WR1, with the vectorclock ", vc)
	request = TM_Message{From: byte(0), To: byte(1), Type: DIFF_REQUEST, VC: *testVC, PageNr: 1}
	diffs = tm.HandleDiffRequest(request)
	assert.Equal(t, diffs.Diffs, []Pair{{vc, tm.GetWriteNoticeListHead(1, 0).Diff.Diffs}},
		"We should recieve the diffs of WR2, with the vectorclock ", vc)
}

func TestTreadMarks_HandleDiffRequest_DiffVCBeforeRequestVC_case1(t *testing.T) {
	tm := SetupHandleDiffRequest()
	testVC := NewVectorclock(3)

	//Lastly we make a timestamp thats ahead of interval timestamp. In this case, we should get no diffs.
	testVC.SetTick(byte(1), 4)
	request := TM_Message{From: byte(0), To: byte(1), Type: DIFF_REQUEST, VC: *testVC, PageNr: 0}
	diffs := tm.HandleDiffRequest(request)
	assert.Len(t, diffs.Diffs, 0,
		"We shouldnt recieve any diffs, because all diffs are before the timestamp")
	request = TM_Message{From: byte(0), To: byte(1), Type: DIFF_REQUEST, VC: *testVC, PageNr: 1}
	diffs = tm.HandleDiffRequest(request)
	assert.Len(t, diffs.Diffs, 0,
		"We shouldnt recieve any diffs, because all diffs are before the timestamp")
}

func TestTreadMarks_HandleDiffRequest_DiffVCBeforeRequestVC_case2(t *testing.T) {
	tm := SetupHandleDiffRequest()
	testVC := NewVectorclock(3)

	//This is just another version of above.
	testVC.SetTick(byte(0), 1)
	testVC.SetTick(byte(1), 3)
	request := TM_Message{From: byte(0), To: byte(1), Type: DIFF_REQUEST, VC: *testVC, PageNr: 0}
	diffs := tm.HandleDiffRequest(request)
	assert.Len(t, diffs.Diffs, 0,
		"We shouldnt recieve any diffs, because all diffs are before the timestamp")
	request = TM_Message{From: byte(0), To: byte(1), Type: DIFF_REQUEST, VC: *testVC, PageNr: 1}
	diffs = tm.HandleDiffRequest(request)
	assert.Len(t, diffs.Diffs, 0,
		"We shouldnt recieve any diffs, because all diffs are before the timestamp")
}

func TestTreadMarks_Barrier(t *testing.T) {
	tm := SetupHandleDiffRequest() //we have ProcId = 1, manager = 0
	cm := NewClientMock()
	tm.IClient = cm

	//intervals from this process has same timestamp (0,3,0) as manager (0,3,0). expect no unseen intervals
	tm.vc = *NewVectorclock(3)
	tm.vc.SetTick(tm.ProcId, 2)
	tm.Barrier(3)
	msg := cm.messages[0]
	testvc := *NewVectorclock(3)
	testvc.SetTick(byte(1), 2)
	assert.Equal(t, testvc, msg.VC)
	assert.Equal(t, byte(1), msg.From)
	assert.Len(t, msg.Intervals, 0)

	//set latest manager timestamp to older than intervals from this process. Expect unseen intervals
	vc := NewVectorclock(3)
	vc.SetTick(byte(0), 2)
	tm.GetIntervalRecordHead(byte(0)).Timestamp = *vc
	tm.Barrier(2)
	fmt.Println(cm.messages)
	msg = cm.messages[1]
	assert.Equal(t, BARRIER_REQUEST, msg.Type)
	assert.Equal(t, byte(0), msg.To)
	assert.Equal(t, tm.ProcId, msg.From)
	assert.Equal(t, 2, msg.Id)
	assert.Len(t, msg.Intervals, 1)
	assert.Len(t, msg.Intervals[0].WriteNotices, len(tm.GetIntervalRecordHead(byte(1)).WriteNotices))
}
