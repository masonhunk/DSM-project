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
	ProcArr []*WriteNoticeRecord
}

type IntervalRecord struct {
	Timestamp		Vectorclock
	WriteNotices []*WriteNoticeRecord
	NextIr      *IntervalRecord
	PrevIr      *IntervalRecord
}

type WriteNoticeRecord struct {
	NextRecord  *WriteNoticeRecord
	PrevRecord  *WriteNoticeRecord
	Interval    *IntervalRecord
	Diff        *Diff
}

type Interval struct{
	Proc byte
	Vt Vectorclock
	WriteNotices []WriteNotice
}

type WriteNotice struct {
	pageNr int
}
type Diff struct {
	Diffs []Pair
}

func CreateDiff(original, new Page) Diff {
	origBytes, err := (*original.vm).ReadBytes(original.addr, (*original.vm).GetPageSize())
	panicOnErr(err)
	newBytes, err := (*original.vm).ReadBytes(original.addr, (*original.vm).GetPageSize())
	panicOnErr(err)
	res := make([]Pair, 0)
	for i, data := range origBytes {
		if newBytes[i] != data {
			res = append(res, Pair{i, newBytes[i]})
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
