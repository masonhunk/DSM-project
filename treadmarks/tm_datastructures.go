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
	GetWritenoticeList(procNr byte, pageNr int) []*WriteNoticeRecord
	PrependWriteNotice(procId byte, wn WriteNotice) *WriteNoticeRecord
	MapWriteNotices(f func(wn *WriteNoticeRecord), pageNr int, procNr byte)
	PrependEmptyWriteNotice(pageNr int, procId byte) *WriteNoticeRecord
	GetCopyset(pageNr int) []int
	SetCopyset(pageNr int, copyset []int)
	GetWriteNoticeListHead(pageNr int, procNr byte) *WriteNoticeRecord
}

type PageArrayEntryInterface interface {
	GetCopyset() []int
	HasCopy() bool
	SetHasCopy(bool bool)
	GetWritenoticeList(procId byte) []*WriteNoticeRecord
	GetWriteNotice(procId byte, pageNr int) *WriteNoticeRecord
	PrependWriteNotice(procId byte, wn WriteNotice) *WriteNoticeRecord
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

type PageArray struct {
	array  map[int]*PageArrayEntry
	nrProc int
}

func (p *PageArray) SetCopyset(pageNr int, copyset []int) {
	p.array[pageNr].copySet = copyset
}

type PageArrayEntry struct {
	copySet                []int
	writeNoticeRecordArray [][]*WriteNoticeRecord
	hascopy                bool
}

type IntervalRecord struct {
	Timestamp    Vectorclock
	WriteNotices []*WriteNoticeRecord
}

type WriteNoticeRecord struct {
	Id          int8
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
	PageNr int
}
type Diff struct {
	Diffs []Pair
}

//Everything that concerns ProcArray
type ProcArray [][]IntervalRecord

func MakeProcArray(nrProc int) ProcArray {
	array := make([][]IntervalRecord, nrProc)
	for i := range array {
		array[i] = make([]IntervalRecord, 0)
	}
	return array
}

func (p ProcArray) GetAllUnseenIntervals(ts Vectorclock) []Interval {
	result := []Interval{}
	for i := 0; i < len(p); i++ {
		result = append(result, p.GetUnseenIntervalsAtProc(ts, byte(i))...)
	}
	return result
}

func (p ProcArray) GetUnseenIntervalsAtProc(ts Vectorclock, procNr byte) []Interval {
	result := []Interval{}
	if len(p[int(procNr)]) == 0 {
		return result
	}
	for _, iRecord := range p[int(procNr)] {
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
	if i >= len(p[procNr]) {
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

func (wn *WriteNoticeRecord) SetDiff(diff *Diff) {
	wn.Diff = diff
}

func (pe *PageArrayEntry) GetWritenoticeList(procId byte) []*WriteNoticeRecord {
	return pe.writeNoticeRecordArray[procId]
}

func (pe *PageArrayEntry) GetWriteNotice(procId byte, index int) *WriteNoticeRecord {
	return pe.writeNoticeRecordArray[procId][index]
}

func (pe *PageArrayEntry) PrependWriteNotice(procId byte, wn WriteNotice) *WriteNoticeRecord {
	wnr := new(WriteNoticeRecord)
	wnr.WriteNotice = wn
	pe.writeNoticeRecordArray[procId] = append([]*WriteNoticeRecord{wnr}, pe.writeNoticeRecordArray[procId]...)
	return wnr
}

func (pe *PageArrayEntry) SetHasCopy(bool bool) {
	pe.hascopy = bool
}

func (pe *PageArrayEntry) GetCopyset() []int {
	return pe.copySet
}

func (pe *PageArrayEntry) HasCopy() bool {
	return pe.hascopy
}

func NewPageArrayEntry(nrProcs int) *PageArrayEntry {
	wnra := make([][]*WriteNoticeRecord, nrProcs+1)
	for i := range wnra {
		wnra[i] = make([]*WriteNoticeRecord, 0)
	}
	return &PageArrayEntry{[]int{0}, wnra, false}
}

//Everything that concerns page arrays

func NewPageArray(nrProc int) *PageArray {
	p := new(PageArray)
	p.array = make(map[int]*PageArrayEntry)
	p.nrProc = nrProc
	return p
}

func (p *PageArray) GetWritenoticeList(procNr byte, pageNr int) []*WriteNoticeRecord {
	return p.array[pageNr].GetWritenoticeList(procNr)
}

func (p *PageArray) SetPageEntry(pageNr int, pe *PageArrayEntry) {
	p.array[pageNr] = pe
}

func (p *PageArray) GetPageEntry(pageNr int) *PageArrayEntry {
	if _, ok := p.array[pageNr]; !ok {
		p.SetPageEntry(pageNr, NewPageArrayEntry(p.nrProc))
	}
	return p.array[pageNr]
}

func (p *PageArray) PrependWriteNotice(procId byte, wn WriteNotice) *WriteNoticeRecord {
	wnr := p.GetPageEntry(wn.PageNr).PrependWriteNotice(procId, wn)
	return wnr
}

func (p *PageArray) MapWriteNotices(f func(wn *WriteNoticeRecord), pageNr int, procNr byte) {
	wns := p.GetWritenoticeList(procNr, pageNr)
	for i := 0; i < len(wns); i++ {
		f(wns[i])
	}
}

func (p *PageArray) PrependEmptyWriteNotice(pageNr int, procId byte) *WriteNoticeRecord {
	return p.GetPageEntry(pageNr).PrependWriteNotice(procId, WriteNotice{})
}

func (p *PageArray) GetCopyset(pageNr int) []int {
	return p.GetPageEntry(pageNr).copySet
}

func (p *PageArray) GetWriteNoticeListHead(pageNr int, procNr byte) *WriteNoticeRecord {
	if len(p.GetWritenoticeList(procNr, pageNr)) < 1 {
		return nil
	}
	return p.GetPageEntry(pageNr).GetWriteNotice(procNr, 0)
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
	Car, Cdr interface{}
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
