package treadmarks

import (
	"DSM-project/memory"
)

type IPage interface {
	Size() int
	Address() int
	ValueAt(offset int) byte
}

type TM_IDataStructures interface {
	PrependEmptyWriteNotice(pageNr int, procId byte) *WriteNoticeRecord
	PrependWriteNotice(procId byte, wn WriteNotice) *WriteNoticeRecord
	AppendIntervalRecord(procNr byte, ir *IntervalRecord)
	PrependIntervalRecord(procNr byte, ir *IntervalRecord)
	GetCopyset(pageNr int) []int
	GetWriteNoticeListHead(pageNr int, procNr byte) *WriteNoticeRecord
	GetIntervalRecordHead(procNr byte) *IntervalRecord
	GetIntervalRecordTail(procNr byte) *IntervalRecord
	GetPageEntry(pageNr int) PageArrayEntry
	SetPageEntry(pageNr int, p PageArrayEntry)
	MapWriteNotices(f func(wn *WriteNoticeRecord), pageNr int, procNr byte)
	MapIntervalRecords(f func(ir *IntervalRecord), procNr byte)
	MapProcArray(f func(p *Pair, procNr byte))
}

type DiffPool []Diff
type ProcArray []Pair
type PageArray map[int]PageArrayEntry

type TM_DataStructures struct {
	diffPool  DiffPool
	procArray ProcArray
	pageArray PageArray
}

func (d *TM_DataStructures) AppendIntervalRecord(procNr byte, ir *IntervalRecord) {
	//var p Pair = d.procArray[procNr]
	//(&p).AppendIntervalRecord(ir)
}

func (d *TM_DataStructures) SetPageEntry(pageNr int, p PageArrayEntry) {
	d.pageArray[pageNr] = p
}

func (d *TM_DataStructures) GetPageEntry(pageNr int) PageArrayEntry {
	return d.pageArray[pageNr]
}

func (d *TM_DataStructures) PrependEmptyWriteNotice(pageNr int, procId byte) *WriteNoticeRecord {
	var p PageArrayEntry = d.pageArray[pageNr]
	return p.PrependWriteNotice(procId)
}

func (d *TM_DataStructures) PrependWriteNotice(procId byte, wn WriteNotice) *WriteNoticeRecord {
	return d.pageArray.PrependWriteNotice(procId, wn)
}

func (d *TM_DataStructures) PrependIntervalRecord(procNr byte, ir *IntervalRecord) {
	d.procArray[procNr].PrependIntervalRecord(ir)
}

func (d *TM_DataStructures) MapProcArray(f func(p *Pair, procNr byte)) {
	for i, p := range d.procArray {
		f(&p, byte(i))
	}
}

func (d *TM_DataStructures) GetIntervalRecordHead(procNr byte) *IntervalRecord {
	return d.procArray[procNr].car.(*IntervalRecord)
}

func (d *TM_DataStructures) GetIntervalRecordTail(procNr byte) *IntervalRecord {
	return d.procArray[procNr].cdr.(*IntervalRecord)
}

func (d *TM_DataStructures) MapWriteNotices(f func(wn *WriteNoticeRecord), pageNr int, procNr byte) {
	wn := d.GetWriteNoticeListHead(pageNr, procNr)
	if wn == nil {
		return
	}
	for {
		f(wn)
		if wn.NextRecord != nil {
			wn = wn.NextRecord
		} else {
			break
		}
	}
}

func (d *TM_DataStructures) MapIntervalRecords(f func(ir *IntervalRecord), procNr byte) {
	interval := d.GetIntervalRecordHead(procNr)
	if interval == nil {
		return
	}
	for {
		f(interval)
		if interval.NextIr != nil {
			interval = interval.NextIr
		} else {
			break
		}
	}
}

func (d *TM_DataStructures) GetCopyset(pageNr int) []int {
	return d.pageArray[pageNr].CopySet
}

func (d *TM_DataStructures) GetWriteNoticeListHead(pageNr int, procNr byte) *WriteNoticeRecord {
	return d.pageArray[pageNr].ProcArr[procNr]
}

type PageArrayEntry struct {
	CopySet []int
	ProcArr map[byte]*WriteNoticeRecord
}

type IntervalRecord struct {
	Timestamp    Vectorclock
	WriteNotices []*WriteNoticeRecord
	NextIr       *IntervalRecord
	PrevIr       *IntervalRecord
}

type WriteNoticeRecord struct {
	WriteNotice WriteNotice
	NextRecord  *WriteNoticeRecord
	PrevRecord  *WriteNoticeRecord
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

func NewPageArrayEntry() *PageArrayEntry {
	return &PageArrayEntry{[]int{}, make(map[byte]*WriteNoticeRecord)}
}

func (p *PageArrayEntry) PrependWriteNotice(procId byte) *WriteNoticeRecord {
	wn := new(WriteNoticeRecord)
	head, ok := p.ProcArr[procId]
	if ok == true {
		head.PrevRecord = wn
		wn.NextRecord = head
	}
	p.ProcArr[procId] = wn

	return wn
}

func (p *PageArray) PrependWriteNotice(procId byte, wn WriteNotice) *WriteNoticeRecord {
	pe, ok := (*p)[int(wn.pageNr)]
	if ok == false {
		(*p)[int(wn.pageNr)] = *NewPageArrayEntry()
		pe = (*p)[int(wn.pageNr)]
	}
	result := pe.PrependWriteNotice(procId)
	result.WriteNotice = wn
	return result
}

func (p *Pair) AppendIntervalRecord(ir *IntervalRecord) {
	if p.cdr == nil {
		p.car = ir
		p.cdr = ir
	} else {
		tail := p.cdr.(*IntervalRecord)
		tail.NextIr = ir
		ir.PrevIr = tail
		p.cdr = ir
	}
}

func (p *Pair) PrependIntervalRecord(ir *IntervalRecord) {
	if p.car == nil {
		p.car = ir
		p.cdr = ir
	} else {
		head := p.car.(*IntervalRecord)
		head.PrevIr = ir
		ir.NextIr = head
		p.car = ir
	}
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
