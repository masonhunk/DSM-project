package treadmarks

import (
	"fmt"
	"sync"
)

type ProcArrayInterface1 interface {
	AddIntervalRecord(procNr byte, ir *IntervalRecord)
	GetIntervalRecord(procNr byte, timestamp Vectorclock) *IntervalRecord
	GetAllUnseenIntervals(ts Vectorclock) []Interval
	GetUnseenIntervalsAtProc(procId byte, ts Vectorclock) []Interval
	MapIntervalRecords(procId byte, f func(ir *IntervalRecord))
	CreateNewInterval(procId byte, timestamp Vectorclock) *IntervalRecord
	GetIntervalRecords(procNr byte) []*IntervalRecord
}

type IntervalRecord struct {
	procId       byte
	Timestamp    Vectorclock
	WriteNotices []*WriteNoticeRecord
}

func (ir *IntervalRecord) AddWriteNoticeRecord(wnr *WriteNoticeRecord) {
	ir.WriteNotices = append(ir.WriteNotices, wnr)
}

var _ ProcArrayInterface1 = &ProcArray1{}

//Everything that concerns interval records
func (ir *IntervalRecord) ToString() string {
	return fmt.Sprint(ir.procId, ir.Timestamp)
}

type Interval struct {
	Proc         byte
	Vt           Vectorclock
	WriteNotices []WriteNotice
}

type ProcArray1 struct {
	array            [][]*IntervalRecord
	managerTimestamp Vectorclock
	locks            []*sync.RWMutex
	dict             map[string]*IntervalRecord
	*sync.RWMutex
}

func NewProcArray(nrProcs int) *ProcArray1 {
	po := new(ProcArray1)
	po.array = make([][]*IntervalRecord, nrProcs+1)
	for i := range po.array {
		po.array[i] = make([]*IntervalRecord, 0)
	}
	po.managerTimestamp = *NewVectorclock(nrProcs)
	po.locks = make([]*sync.RWMutex, nrProcs+1)
	for i := range po.locks {
		po.locks[i] = new(sync.RWMutex)
	}
	po.dict = make(map[string]*IntervalRecord)
	po.RWMutex = new(sync.RWMutex)
	return po
}

func (po *ProcArray1) AddIntervalRecord(procId byte, ir *IntervalRecord) {
	ir.procId = procId
	po.LockProc(procId)
	po.array[int(procId)] = append(po.array[int(procId)], ir)
	po.UnlockProc(procId)
	po.Lock()
	po.dict[fmt.Sprint(procId, ir.Timestamp)] = ir
	po.Unlock()
}

func (po *ProcArray1) CreateNewInterval(procId byte, timestamp Vectorclock) *IntervalRecord {
	interval := &IntervalRecord{
		procId:       procId,
		Timestamp:    timestamp,
		WriteNotices: make([]*WriteNoticeRecord, 0),
	}
	return interval
}

func (po *ProcArray1) GetIntervalRecord(procId byte, timestamp Vectorclock) *IntervalRecord {
	po.RLock()
	defer po.RUnlock()
	return po.dict[fmt.Sprint(procId, timestamp)]
}

func (po *ProcArray1) GetIntervalRecords(procId byte) []*IntervalRecord {
	po.RLock()
	defer po.RUnlock()
	return po.array[int(procId)]
}

func (po *ProcArray1) GetAllUnseenIntervals(ts Vectorclock) []Interval {
	result := make([]Interval, 0)
	for proc := range po.array {
		result = append(result, po.GetUnseenIntervalsAtProc(byte(proc), ts)...)
	}
	return result
}

func (po *ProcArray1) GetUnseenIntervalsAtProc(procId byte, ts Vectorclock) []Interval {
	result := make([]Interval, 0)
	po.RLockProc(procId)
	array := po.array[int(procId)]
	var thisTs Vectorclock
	for i := len(array) - 1; i >= 0; i-- {
		thisTs = array[i].Timestamp
		if thisTs.Equals(ts) {
			break
		} else if thisTs.IsBefore(ts) {
			break
		}
		wns := make([]WriteNotice, len(array[i].WriteNotices))
		wnrs := array[i].WriteNotices
		for i, wnr := range wnrs {
			wns[i] = WriteNotice{PageNr: wnr.pageNr}
		}
		interval := Interval{
			Proc:         procId,
			Vt:           thisTs,
			WriteNotices: wns,
		}
		result = append(result, interval)
	}
	po.RUnlockProc(procId)
	return result
}

func (po *ProcArray1) LockProc(procId byte) {
	if po.locks[int(procId)] == nil {
		po.locks[int(procId)] = new(sync.RWMutex)
	}
	po.locks[int(procId)].Lock()
}

func (po *ProcArray1) RLockProc(procId byte) {
	if po.locks[int(procId)] == nil {
		po.locks[int(procId)] = new(sync.RWMutex)
	}
	po.locks[int(procId)].RLock()
}

func (po *ProcArray1) UnlockProc(procId byte) {
	if po.locks[int(procId)] == nil {
		po.locks[int(procId)] = new(sync.RWMutex)
	}
	po.locks[int(procId)].Unlock()
}
func (po *ProcArray1) RUnlockProc(procId byte) {
	if po.locks[int(procId)] == nil {
		po.locks[int(procId)] = new(sync.RWMutex)
	}
	po.locks[int(procId)].RUnlock()
}

func (po *ProcArray1) MapIntervalRecords(procId byte, f func(ir *IntervalRecord)) {
	po.LockProc(procId)
	for _, int := range po.array[int(procId)] {
		f(int)
	}
	po.UnlockProc(procId)
}
