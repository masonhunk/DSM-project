package datastructure

import (
	"sync"
	"DSM-project/treadmarks"
	"fmt"
)

type PageArrayEntryInterface interface{
	AddWriteNoticeRecord(procId byte, wnr WriteNoticeRecordInterface) bool
	GetWriteNoticeRecordById(id string) WriteNoticeRecordInterface
	GetWriteNoticeRecord(pageNr int, procId byte, timestamp treadmarks.Timestamp) WriteNoticeRecordInterface
	GetHasCopy() bool
	SetHasCopy(b bool)
	GetCopySet() []byte
	AddToCopySet(procId byte)
	GetMissingDiffs() []DiffDescription
	Lock()
	Unlock()
	RLock()
	RUnlock()
}

type PageArrayEntry struct{
	wnrLists [][]WriteNoticeRecordInterface
	wnrMap map[string]WriteNoticeRecordInterface
	hasCopy bool
	copySet []byte
	lock *sync.RWMutex
}

func NewPageArrayEntry(nrProcs int) *PageArrayEntry{
	p := new(PageArrayEntry)
	p.wnrLists = make([][]WriteNoticeRecordInterface, nrProcs)
	for i := range p.wnrLists{
		p.wnrLists[i] = make([]WriteNoticeRecordInterface, 0)
	}
	p.hasCopy = false
	p.copySet = []byte{0}
	p.lock = new(sync.RWMutex)
	p.wnrMap = make(map[string]WriteNoticeRecordInterface)
	return p
}

func (pe *PageArrayEntry) AddWriteNoticeRecord(procId byte, wnr WriteNoticeRecordInterface) bool {
	pe.Lock()
	defer pe.Unlock()
	expanded := false
	l := len(pe.wnrLists[int(procId)-1])
	c := cap(pe.wnrLists[int(procId)-1])
	if l+1 > c{
		temp := make([]WriteNoticeRecordInterface, l, (l+1)*2)
		copy(temp, pe.wnrLists[int(procId)-1])
		pe.wnrLists[int(procId)-1] = temp
		for i, w := range pe.wnrLists[int(procId)-1]{
			w.GetId()
			pe.wnrMap[w.GetId()] = pe.wnrLists[int(procId)-1][i]
		}
		expanded = true
	}
	pe.wnrLists[int(procId)-1] = append(pe.wnrLists[int(procId)-1], wnr)
	pe.wnrMap[wnr.GetId()] = pe.wnrLists[int(procId)-1][len(pe.wnrLists[int(procId)-1])-1]
	return expanded
}

func (pe *PageArrayEntry) GetWriteNoticeRecordById(id string) WriteNoticeRecordInterface {
	pe.RLock()
	defer pe.RUnlock()
	return pe.wnrMap[id]
}

func (pe *PageArrayEntry) GetWriteNoticeRecord(pageNr int, procId byte, timestamp treadmarks.Timestamp) WriteNoticeRecordInterface {
	return pe.GetWriteNoticeRecordById(fmt.Sprint("wn_", pageNr, procId, timestamp))
}

func (pe *PageArrayEntry) GetHasCopy() bool {
	return pe.hasCopy
}

func (pe *PageArrayEntry) SetHasCopy(b bool) {
	pe.hasCopy = b
}

func (pe *PageArrayEntry) GetCopySet() []byte {
	return pe.copySet
}

func (pe *PageArrayEntry) AddToCopySet(procId byte) {
	pe.copySet = append(pe.copySet, procId)
}

func (pe *PageArrayEntry) GetMissingDiffs() []DiffDescription{
	missing := make([]DiffDescription, 0)
	for i, wnl := range pe.wnrLists{
		for _, wn := range wnl {
			if wn.GetDiff() == nil{
				missing = append(missing, DiffDescription{ProcId: byte(i+1), Timestamp: wn.GetTimestamp()})
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return missing
}

func (pe *PageArrayEntry) Lock() {
	pe.lock.Lock()
}

func (pe *PageArrayEntry) Unlock() {
	pe.lock.Unlock()
}

func (pe *PageArrayEntry) RLock() {
	pe.lock.RLock()
}

func (pe *PageArrayEntry) RUnlock() {
	pe.lock.RUnlock()
}
