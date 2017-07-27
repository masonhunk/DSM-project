package treadmarks

import (
	"errors"
	"fmt"
	"sync"
)

type PageArrayInterface1 interface {
	AddWriteNoticeRecord(procId byte, pageNr int, wnr *WriteNoticeRecord)
	GetWritenoticeRecords(procId byte, pageNr int) []*WriteNoticeRecord
	GetWritenoticeRecord(procId byte, pageNr int, timestamp Vectorclock) *WriteNoticeRecord
	CreateNewWritenoticeRecord(procId byte, pageNr int, int *IntervalRecord) *WriteNoticeRecord
	SetDiff(pageNr int, diff DiffDescription)
	SetDiffs(pageNr int, diffs []DiffDescription)
	MapWriteNotices(f func(wn *WriteNoticeRecord), pageNr int, procId byte)
	GetCopyset(pageNr int) []byte
	AddToCopyset(procId byte, pageNr int)
	HasCopy(pageNr int) bool
	SetHasCopy(pageNr int, b bool)
	NewDiffIterator(pageNr int) *DiffIterator
	GetMissingDiffTimestamps(pageNr int) []Pair
}

type PageArrayEntryInterface1 interface {
	GetCopyset() []byte
	AddToCopyset(procId byte)
	HasCopy() bool
	SetHasCopy(bool bool)
	AddWritenoticeRecord(procId byte, wnr *WriteNoticeRecord)
	CreateWritenoticeRecord(procId byte, int *IntervalRecord) *WriteNoticeRecord
	GetWriteNoticeRecord(procId byte, timestamp Vectorclock) *WriteNoticeRecord
	SetDiff(diff DiffDescription)
	SetDiffs(diffs []DiffDescription)
	GenerateDiffRequests(pageNr int, group *sync.WaitGroup) []TM_Message
	NewDiffIterator() *DiffIterator
	GetMissingDiffTimestamps() []Pair
	sync.Locker
}

type WriteNoticeRecord struct {
	Id          int
	WriteNotice WriteNotice
	pageNr      int
	Interval    *IntervalRecord //Make this hidden
	Diff        *Diff
}

type WriteNotice struct {
	PageNr int
}

//Everything that concerns WriteNoticeRecords
func (wnr *WriteNoticeRecord) ToString() string {
	return fmt.Sprint(wnr.pageNr, wnr.GetTimestamp().Value)
}

func (wnr *WriteNoticeRecord) GetId() int {
	return wnr.Id
}

func (wnr *WriteNoticeRecord) GetTimestamp() *Vectorclock {
	if wnr.Interval != nil {
		return &wnr.Interval.Timestamp
	}
	return &Vectorclock{}
}

type PageArray1 struct {
	nrProcs int
	array   map[int]*PageArrayEntry1
	*sync.RWMutex
	procId *byte
}

func NewPageArray1(procId *byte, nrProcs int) *PageArray1 {
	p := new(PageArray1)
	p.nrProcs = nrProcs
	p.procId = procId
	p.array = make(map[int]*PageArrayEntry1)
	p.RWMutex = new(sync.RWMutex)
	return p
}

func (p *PageArray1) AddWriteNoticeRecord(procId byte, pageNr int, wnr *WriteNoticeRecord) {
	pe := p.getEntry(pageNr)
	pe.Lock()
	pe.AddWritenoticeRecord(procId, wnr)
	pe.Unlock()
}

func (p *PageArray1) SetHasCopy(pageNr int, b bool) {
	pe := p.getEntry(pageNr)
	pe.Lock()
	pe.SetHasCopy(b)
	pe.Unlock()
}

func (p *PageArray1) CreateNewWritenoticeRecord(procId byte, pageNr int, int *IntervalRecord) *WriteNoticeRecord {
	wn := WriteNotice{PageNr: pageNr}
	wnr := &WriteNoticeRecord{
		Id:          0,
		WriteNotice: wn,
		pageNr:      pageNr,
		Interval:    int,
	}
	int.AddWriteNoticeRecord(wnr)
	return wnr
}

func (p *PageArray1) GetWritenoticeRecords(procId byte, pageNr int) []*WriteNoticeRecord {
	pe := p.getEntry(pageNr)
	pe.RLock()
	wnrs := pe.writeNoticeArray[int(procId)-1]
	pe.RUnlock()
	return wnrs
}

func (p *PageArray1) GetWritenoticeRecord(procId byte, pageNr int, timestamp Vectorclock) *WriteNoticeRecord {
	pe := p.getEntry(pageNr)
	pe.RLock()
	wnr := pe.GetWriteNoticeRecord(procId, timestamp)
	pe.RUnlock()
	return wnr
}

func (p *PageArray1) SetDiff(pageNr int, diff DiffDescription) {
	pe := p.getEntry(pageNr)
	pe.RLock()
	wnr := pe.GetWriteNoticeRecord(diff.ProcId, diff.Timestamp)
	pe.RUnlock()
	wnr.Diff = &diff.Diff
}

func (p *PageArray1) SetDiffs(pageNr int, diffs []DiffDescription) {
	pe := p.getEntry(pageNr)
	pe.RLock()
	for _, diff := range diffs {
		wnr := pe.GetWriteNoticeRecord(diff.ProcId, diff.Timestamp)
		newDiff := diff.Diff
		if wnr == nil {
			panicOnErr(errors.New("Got diff response with nil diff!"))
		}
		wnr.Diff = &newDiff
	}
	pe.RUnlock()
}

func (p *PageArray1) MapWriteNotices(f func(wn *WriteNoticeRecord), pageNr int, procNr byte) {
	pe := p.getEntry(pageNr)
	pe.RLock()
	for _, wnr := range pe.writeNoticeArray[procNr] {
		f(wnr)
	}
	pe.RUnlock()
}

func (p *PageArray1) GetCopyset(pageNr int) []byte {
	pe := p.getEntry(pageNr)
	return pe.GetCopyset()
}

func (p *PageArray1) AddToCopyset(procId byte, pageNr int) {
	pe := p.getEntry(pageNr)
	pe.Lock()
	pe.copySet = append(pe.copySet, procId)
	pe.Unlock()
}

func (p *PageArray1) getEntry(pageNr int) *PageArrayEntry1 {
	p.Lock()
	pe, ok := p.array[pageNr]
	if !ok {
		pe = NewPageArrayEntry1(p.procId, p.nrProcs, pageNr)
		p.array[pageNr] = pe
	}
	p.Unlock()
	return pe
}

func (p *PageArray1) HasCopy(pageNr int) bool {
	p.Lock()
	pe, ok := p.array[pageNr]
	if !ok {
		pe = NewPageArrayEntry1(p.procId, p.nrProcs, pageNr)
		p.array[pageNr] = pe
	}
	p.Unlock()
	pe.RLock()
	result := pe.HasCopy()
	pe.RUnlock()
	return result
}

func (p *PageArray1) GetMissingDiffTimestamps(pageNr int) []Pair {
	pe := p.getEntry(pageNr)
	pe.RLock()
	defer pe.RUnlock()
	return pe.GetMissingDiffTimestamps()
}

func (p *PageArray1) NewDiffIterator(pageNr int) *DiffIterator {
	return p.getEntry(pageNr).NewDiffIterator()
}

type PageArrayEntry1 struct {
	pageNr                   int
	procId                   *byte
	nrProcs                  int
	copySet                  []byte
	hasCopy                  bool
	writeNoticeArray         [][]*WriteNoticeRecord
	writeNoticeArrayVCLookup map[string]*WriteNoticeRecord
	*sync.RWMutex
}

func NewPageArrayEntry1(procId *byte, nrProcs int, pageNr int) *PageArrayEntry1 {
	pe := new(PageArrayEntry1)
	pe.pageNr = pageNr
	pe.procId = procId
	pe.nrProcs = nrProcs
	pe.copySet = []byte{byte(0)}
	pe.writeNoticeArray = make([][]*WriteNoticeRecord, nrProcs)
	pe.writeNoticeArrayVCLookup = make(map[string]*WriteNoticeRecord)
	pe.RWMutex = new(sync.RWMutex)
	for i := range pe.writeNoticeArray {
		pe.writeNoticeArray[i] = make([]*WriteNoticeRecord, 0)
	}
	return pe
}

func (pe *PageArrayEntry1) GetCopyset() []byte {
	return pe.copySet
}

func (pe *PageArrayEntry1) AddToCopyset(procId byte) {
	pe.copySet = append(pe.copySet, procId)
}

func (pe *PageArrayEntry1) HasCopy() bool {
	return pe.hasCopy
}

func (pe *PageArrayEntry1) SetHasCopy(hasCopy bool) {
	pe.hasCopy = hasCopy
}

func (pe *PageArrayEntry1) AddWritenoticeRecord(procId byte, wnr *WriteNoticeRecord) {
	wnr.pageNr = pe.pageNr
	newSlice := make([]*WriteNoticeRecord, len(pe.writeNoticeArray[int(procId)-1])+1)
	copy(newSlice[1:], pe.writeNoticeArray[int(procId)-1])
	newSlice[0] = wnr
	pe.writeNoticeArray[int(procId)-1] = newSlice
	pe.writeNoticeArrayVCLookup[wnr.ToString()] = wnr
}

func (pe *PageArrayEntry1) CreateWritenoticeRecord(procId byte, int *IntervalRecord) *WriteNoticeRecord {
	wnr := new(WriteNoticeRecord)
	wnr.Interval = int
	wnr.pageNr = pe.pageNr
	return wnr
}

func (pe *PageArrayEntry1) GetWriteNoticeRecord(procId byte, timestamp Vectorclock) *WriteNoticeRecord {
	wr := pe.writeNoticeArrayVCLookup[fmt.Sprint(pe.pageNr, timestamp.Value)]
	return wr
}

func (pe *PageArrayEntry1) SetDiff(diff DiffDescription) {
	newDiff := diff.Diff
	pe.GetWriteNoticeRecord(diff.ProcId, diff.Timestamp).Diff = &newDiff
}

func (pe *PageArrayEntry1) SetDiffs(diffs []DiffDescription) {
	for _, diff := range diffs {
		pe.SetDiff(diff)
	}
}

func (pe *PageArrayEntry1) GetMissingDiffTimestamps() []Pair {
	//First we check if we have the page already or need to request a copy.
	//First we find the start timestamps
	pe.RLock()
	defer pe.RUnlock()
	ProcStartTS := make([]Vectorclock, pe.nrProcs)
	ProcEndTS := make([]Vectorclock, pe.nrProcs)
	for i := 0; i < pe.nrProcs; i++ {
		wnrl := pe.writeNoticeArray[byte(i)]
		if len(wnrl) < 1 {
			continue
		}
		ProcStartTS[i] = *wnrl[0].GetTimestamp()
		for _, wnr := range wnrl {
			if wnr.Diff != nil {
				break
			}
			ProcEndTS[i] = wnr.Interval.Timestamp
		}
	}

	//Then we "merge" the different intervals
	for i := 0; i < pe.nrProcs; i++ {
		if ProcStartTS[i].Value == nil {
			continue
		}
		for j := i; j < pe.nrProcs; j++ {
			if ProcStartTS[j].Value == nil {
				continue
			}
			if ProcStartTS[i].IsAfter(ProcStartTS[j]) {
				if ProcEndTS[j].Value != nil && (ProcEndTS[i].Value == nil || ProcEndTS[i].IsAfter(ProcEndTS[j])) {
					ProcEndTS[i] = ProcEndTS[j]
				}
				ProcEndTS[j] = Vectorclock{}
			} else if ProcStartTS[j].IsAfter(ProcStartTS[i]) {
				if ProcEndTS[i].Value != nil && (ProcEndTS[j].Value == nil || ProcEndTS[j].IsAfter(ProcEndTS[i])) {
					ProcEndTS[j] = ProcEndTS[i]
				}
				ProcEndTS[i] = Vectorclock{}
			}
		}
	}
	//Then we build the messages
	pairs := make([]Pair, pe.nrProcs)

	for i := 0; i < pe.nrProcs; i++ {
		if ProcStartTS[i].Value == nil || ProcEndTS[i].Value == nil {
			continue
		}
		pairs[i] = Pair{ProcStartTS[i], ProcEndTS[i]}
	}

	return pairs
}

type DiffIterator struct {
	index  []int
	order  []byte
	pageNr int
	lock   *sync.RWMutex
	pe     *PageArrayEntry1
}

func (pe *PageArrayEntry1) NewDiffIterator() *DiffIterator {
	//First we populate the iterator.
	di := new(DiffIterator)
	di.index = make([]int, len(pe.writeNoticeArray))
	for i := range di.index {
		di.index[i] = len(pe.writeNoticeArray[i]) - 1
	}
	di.order = make([]byte, 0)
	di.pageNr = pe.pageNr
	di.lock = pe.RWMutex
	di.lock.Lock()

	di.pe = pe
	for i := range di.index {
		di.Insert(byte(i + 1))
	}
	return di
}

func (di *DiffIterator) Next() *Diff {
	var proc byte
	if len(di.order) > 1 {
		proc, di.order = di.order[0], di.order[1:]

	} else if len(di.order) == 1 {
		proc, di.order = di.order[0], make([]byte, 0)
	} else {
		return nil
	}
	index := di.index[int(proc)-1]
	wnrl := di.pe.writeNoticeArray[int(proc)-1]
	wnr := wnrl[index]
	di.index[int(proc)-1] = index - 1
	di.Insert(proc)

	return wnr.Diff
}

func (di *DiffIterator) Close() {
	di.lock.Unlock()
}

func (di *DiffIterator) Insert(proc byte) {
	var this *WriteNoticeRecord
	var that *WriteNoticeRecord
	i := int(proc) - 1
	if di.index[i] < 0 {
		// This proc dont have any writenotices.
		return
	}
	length := len(di.order)
	this = di.pe.writeNoticeArray[i][0]
	for j := range di.index {
		if j == length {
			//If we are at the end of the order array, we append and break.
			di.order = append(di.order, proc)
			length++
			break
		}
		if len(di.pe.writeNoticeArray[di.order[j]]) < di.index[di.order[j]]+1 {
			//If that in the order array doesnt have any writenotices, we will insert this before that.
			di.order = append(di.order, byte(0))
			copy(di.order[j+1:], di.order[j:])
			di.order[j] = proc
			break
		}
		that = di.pe.writeNoticeArray[di.order[j]][di.index[di.order[j]]]
		if this.GetTimestamp().IsBefore(*that.GetTimestamp()) {
			// If this timestamp is before that, we insert this before that.
			di.order = append(di.order, byte(0))
			copy(di.order[j+1:], di.order[j:])
			di.order[j] = proc
			break
		}
		if !this.GetTimestamp().IsAfter(*that.GetTimestamp()) && proc < di.order[j] {
			// If this timestamp is not after that, and this proc is smaller than that proc, we insert it before that.
			di.order = append(di.order, byte(0))
			copy(di.order[j+1:], di.order[j:])
			di.order[j] = proc
			break
		}
	}
}

type DiffDescription struct {
	ProcId    byte
	Timestamp Vectorclock
	Diff      Diff
}

func CreateDiffDescription(procId byte, wnr *WriteNoticeRecord) DiffDescription {
	return DiffDescription{
		ProcId:    procId,
		Timestamp: wnr.Interval.Timestamp,
		Diff:      *wnr.Diff, //TODO make Diff in the writenoticerecord a list of pairs, isntead of storing it in its own struct.
	}
}
