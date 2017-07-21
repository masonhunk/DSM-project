package treadmarks

import (
	"DSM-project/memory"
	"sync"
)

type IPage interface {
	Size() int
	Address() int
	ValueAt(offset int) byte
}

type TM_IDataStructures interface {
	sync.Locker
	PageArrayInterface
	ProcArrayInterface
}

type PageArrayInterface interface {
	SetPageEntry(pageNr int, p *PageArrayEntry)
	GetPageEntry(pageNr int) *PageArrayEntry
	GetWritenotices(procNr byte, pageNr int) []WriteNoticeRecord
	PrependWriteNotice(procId byte, wn WriteNotice) *WriteNoticeRecord
	MapWriteNotices(f func(wn *WriteNoticeRecord), pageNr int, procNr byte)
	PrependEmptyWriteNotice(pageNr int, procId byte) *WriteNoticeRecord
	GetCopyset(pageNr int) []int
	GetWriteNoticeListHead(pageNr int, procNr byte) *WriteNoticeRecord
}

type PageArrayEntryInterface interface {
	PrependWriteNotice(procId byte) *WriteNoticeRecord
}

type ProcArrayInterface interface {
	GetIntervalRecordHead(procNr byte) *IntervalRecord
	GetIntervalRecordTail(procNr byte) *IntervalRecord
	PrependIntervalRecord(procNr byte, ir *IntervalRecord)
	GetIntervalRecord(procNr byte, i int) *IntervalRecord
	GetAllUnseenIntervals(ts Vectorclock) []Interval
	GetUnseenIntervalsAtProc(ts Vectorclock, procNr byte) []Interval
	MapIntervalRecords(f func(ir *IntervalRecord), procNr byte)
}

type TM_DataStructures struct {
	*sync.RWMutex
	ProcArray
	PageArray
}

type IntervalRecord struct {
	Timestamp    Vectorclock
	WriteNotices []*WriteNoticeRecord
}

type WriteNoticeRecord struct {
	id          int8
	WriteNotice WriteNotice
	Interval    *IntervalRecord
	Diff        *Diff
}

type Interval struct {
	Proc         byte
	Vt           Vectorclock
	WriteNotices []WriteNotice
}

type WriteNotice struct {
	pageNr int
}
type Diff struct {
	Diffs []Pair
}

//Everything that concerns ProcArray
type ProcArray map[byte][]IntervalRecord

func (p ProcArray) GetAllUnseenIntervals(ts Vectorclock) []Interval {
	result := []Interval{}
	for i := 0; i < len(p); i++ {
		result = append(result, p.GetUnseenIntervalsAtProc(ts, byte(i))...)
	}
	return result
}

func (p ProcArray) GetUnseenIntervalsAtProc(ts Vectorclock, procNr byte) []Interval {
	result := []Interval{}
	for _, iRecord := range p[procNr] {
		// if this record has older ts than the requester, break
		if iRecord.Timestamp.IsBefore(ts) || iRecord.Timestamp.Equals(ts) {
			break
		}
		i := Interval{
			Proc: procNr,
			Vt:   iRecord.Timestamp,
		}
		for _, wn := range iRecord.WriteNotices {
			i.WriteNotices = append(i.WriteNotices, wn.WriteNotice)
		}
		result = append(result, i)
	}
	return result
}

func (p ProcArray) PrependIntervalRecord(procNr byte, ir *IntervalRecord) {
	p[procNr] = append([]IntervalRecord{*ir}, p[procNr]...)
}

func (p ProcArray) GetIntervalRecordHead(procNr byte) *IntervalRecord {
	return p.GetIntervalRecord(procNr, 0)
}

func (p ProcArray) GetIntervalRecord(procNr byte, i int) *IntervalRecord {
	if _, ok := p[procNr]; !ok {
		return nil
	} else if i >= len(p[procNr]) {
		return nil
	}
	return &p[procNr][i]
}

func (p ProcArray) MapIntervalRecords(f func(ir *IntervalRecord), procNr byte) {
	for _, interval := range p[procNr] {
		f(&interval)
	}
}

func (p ProcArray) GetIntervalRecordTail(procNr byte) *IntervalRecord {
	return &p[procNr][len(p[procNr])-1]
}

//Everything that concerns page entries
type PageArrayEntry struct {
	CopySet                []int
	WriteNoticeRecordArray map[byte][]WriteNoticeRecord
}

func NewPageArrayEntry() *PageArrayEntry {
	return &PageArrayEntry{[]int{0}, make(map[byte][]WriteNoticeRecord)}
}

func (pe *PageArrayEntry) PrependWriteNoticeOnPageArrayEntry(procId byte) *WriteNoticeRecord {
	if pe.WriteNoticeRecordArray == nil {
		pe.WriteNoticeRecordArray = make(map[byte][]WriteNoticeRecord)
	}
	wnr := new(WriteNoticeRecord)
	pe.WriteNoticeRecordArray[procId] = append([]WriteNoticeRecord{*wnr}, pe.WriteNoticeRecordArray[procId]...)
	return &pe.WriteNoticeRecordArray[procId][0]
}

//Everything that concerns page arrays
type PageArray map[int]*PageArrayEntry

func (p PageArray) SetPageEntry(pageNr int, pe *PageArrayEntry) {
	p[pageNr] = pe
}

func (p PageArray) GetPageEntry(pageNr int) *PageArrayEntry {
	if _, ok := p[pageNr]; !ok {
		p.SetPageEntry(pageNr, &PageArrayEntry{})
	}
	return p[pageNr]
}

func (p PageArray) GetWritenotices(procNr byte, pageNr int) []WriteNoticeRecord {
	return p[pageNr].WriteNoticeRecordArray[procNr]
}

func (p PageArray) PrependWriteNotice(procId byte, wn WriteNotice) *WriteNoticeRecord {
	wnr := p.GetPageEntry(wn.pageNr).PrependWriteNoticeOnPageArrayEntry(procId)
	wnr.WriteNotice = wn
	return wnr
}

func (p PageArray) MapWriteNotices(f func(wn *WriteNoticeRecord), pageNr int, procNr byte) {
	wns := p.GetPageEntry(pageNr).WriteNoticeRecordArray[procNr]
	for i := 0; i < len(wns); i++ {
		f(&wns[i])
	}
}

func (p PageArray) PrependEmptyWriteNotice(pageNr int, procId byte) *WriteNoticeRecord {
	return p.GetPageEntry(pageNr).PrependWriteNoticeOnPageArrayEntry(procId)
}

func (p PageArray) GetCopyset(pageNr int) []int {
	return p[pageNr].CopySet
}

func (p PageArray) GetWriteNoticeListHead(pageNr int, procNr byte) *WriteNoticeRecord {

	if len(p[pageNr].WriteNoticeRecordArray[procNr]) < 1 {
		return nil
	}
	return &p[pageNr].WriteNoticeRecordArray[procNr][0]
}

func CreateDiff(original, new []byte) Diff {
	res := make([]Pair, 0)
	for i, data := range original {
		if new[i] != data {
			res = append(res, Pair{i, new[i]})
		}
	}
	return Diff{res}
}

type Pair struct {
	car, cdr interface{}
}

type Page struct {
	vm   *memory.VirtualMemory
	addr int
}

func (p Page) Size() int {
	return (*p.vm).GetPageSize()
}

func (p *Page) Address() int {
	return p.addr
}

func (p *Page) ValueAt(offset int) byte {
	(*p.vm).AccessRightsDisabled(true)
	res, _ := (*p.vm).Read(p.addr + offset)
	(*p.vm).AccessRightsDisabled(false)
	return res
}
