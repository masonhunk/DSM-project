package treadmarks

import (
	"DSM-project/memory"
)

type IPage interface {
	Size() int
	Address() int
	ValueAt(offset int) byte
}

type DiffPool []Diff
type ProcArray []Pair
type PageArray map[int]PageArrayEntry

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
	pe,ok := (*p)[int(wn.pageNr)]
	if ok == false{
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
