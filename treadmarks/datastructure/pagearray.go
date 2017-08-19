package datastructure

import (
	"DSM-project/treadmarks"
	"sync"
)

type PageArrayInterface interface{
	AddWriteNotices(procId byte, timestamp treadmarks.Timestamp, pages []int, intervalId string)
	CreateWriteNotices(timestamp treadmarks.Timestamp, intervalId string)
	SetDiffs(pageNr int, diffDesses []DiffDescription)
	GetHasCopy(pageNr int) bool
	SetHasCopy(pageNr int, b bool)
	GetCopySet(pageNr int) []byte
	AddToCopySet(pageNr int, procId byte)
	SetTwin(pageNr int, data []byte)
	HasTwin(pageNr int) []byte
	Lock(pageNr int)
	Unlock(pageNr int)
	RLock(pageNr int)
	RUnlock(pageNr int)
}

type PageArray struct{
	myId byte
	entries []PageArrayEntryInterface
	twins map[int][]byte
	twinLock *sync.RWMutex
}

func (p *PageArray) AddWriteNotices(procId byte, timestamp treadmarks.Timestamp, pages []int, intervalId string) {
	for _, pageNr := range pages{
		wnr := NewWriteNoticeRecord(pageNr, procId, timestamp, nil)
		p.entries[pageNr].AddWriteNoticeRecord(procId, wnr)
	}
}

func (p *PageArray) CreateWriteNotice(timestamp treadmarks.Timestamp, pageNr int){
	if p.entries[pageNr].GetWriteNoticeRecord(pageNr, p.myId, timestamp) == nil{
		wnr := NewWriteNoticeRecord(pageNr, p.myId, timestamp, nil)
		p.entries[pageNr].AddWriteNoticeRecord(p.myId, wnr)
	}
}

func (p *PageArray) CreateWriteNotices(timestamp treadmarks.Timestamp, intervalId string) {
	p.twinLock.Lock()
	defer p.twinLock.Unlock()
	for pageNr := range p.twins {
		p.CreateWriteNotice(timestamp, pageNr)
	}
}

func (p *PageArray) GetWriteNotice(pageNr int, procId byte, timestamp treadmarks.Timestamp) WriteNoticeRecordInterface {
	p.RLock(pageNr)
	defer p.RUnlock(pageNr)
	return p.entries[pageNr].GetWriteNoticeRecord(pageNr, procId, timestamp)
}

func (p *PageArray) SetDiffs(pageNr int, diffDesses []DiffDescription) {
	for _, diffDes := range diffDesses{
		p.entries[pageNr].GetWriteNoticeRecord(pageNr, diffDes.ProcId, diffDes.Timestamp).SetDiff(diffDes.Diff)
	}
}

func (p *PageArray) GetHasCopy(pageNr int) bool {
	panic("implement me")
}

func (p *PageArray) SetHasCopy(pageNr int, b bool) {
	panic("implement me")
}

func (p *PageArray) GetCopySet(pageNr int) []byte {
	panic("implement me")
}

func (p *PageArray) AddToCopySet(pageNr int, procId byte) {
	panic("implement me")
}

func (p *PageArray) SetTwin(pageNr int, data []byte) {
	panic("implement me")
}

func (p *PageArray) GetTwin(pageNr int) []byte {
	panic("implement me")
}

func (p *PageArray) Lock(pageNr int) {
	panic("implement me")
}

func (p *PageArray) Unlock(pageNr int) {
	panic("implement me")
}

func (p *PageArray) RLock(pageNr int) {
	panic("implement me")
}

func (p *PageArray) RUnlock(pageNr int) {
	panic("implement me")
}





