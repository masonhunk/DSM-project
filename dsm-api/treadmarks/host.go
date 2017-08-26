package treadmarks

import (
	"DSM-project/dsm-api"
	"sync"
	"bytes"
	"encoding/gob"
	"DSM-project/memory"
)


type TreadmarksApi struct{
	memory                         memory.VirtualMemory
	myId, nrProcs                  uint8
	memSize, pageByteSize, nrPages int
	pagearray                      map[int16]pageArrayEntry
	twinMap                        map[int16][]byte
	procarray                      [][]IntervalRecord
	locks                          []Lock
	channel                        chan bool
	barriers                       []chan uint8
	sessionId                      int
	timestamp                      Timestamp
	in, out                        chan []byte
	enc                            *gob.Encoder
	encBuff                        *bytes.Buffer
	encLock                        *sync.Mutex

}

var _ dsm_api.DSMApiInterface = new(TreadmarksApi)

func NewTreadmarksApi(memSize, pageByteSize int, nrProcs, nrLocks, nrBarriers  uint8) (*TreadmarksApi, error) {
	var err error
	t := new(TreadmarksApi)
	t.memory = memory.NewVmem(memSize, pageByteSize)
	t.nrPages = memSize / pageByteSize + Min(memSize % pageByteSize, 1)
	t.memSize, t.pageByteSize, t.nrProcs = memSize, pageByteSize, nrProcs
	t.pagearray = make(map[int16]pageArrayEntry)
	t.procarray = make([][]IntervalRecord, nrProcs)
	for i := range t.procarray{
		t.procarray[i] = make([]IntervalRecord,0)
	}
	t.locks = make([]Lock, nrLocks)
	t.channel = make(chan bool, 1)
	t.barriers = make([]chan uint8, nrBarriers)
	t.timestamp = NewTimestamp(nrProcs)
	t.encBuff = bytes.NewBuffer([]byte{})
	t.enc = gob.NewEncoder(t.encBuff)
	t.encLock = new(sync.Mutex)
	return t, err
}
//----------------------------------------------------------------//
//              Functions defined by the interface                //
//----------------------------------------------------------------//

func (t *TreadmarksApi) Initialize(port int) error {
	panic("implement me")
}

func (t *TreadmarksApi) Join(ip string, port int) error {
	panic("implement me")
}

func (t *TreadmarksApi) Shutdown() error {
	panic("implement me")
}

func (t *TreadmarksApi) Read(addr int) (byte, error) {
	panic("implement me")
}

func (t *TreadmarksApi) Write(addr int, val byte) error {
	panic("implement me")
}

func (t *TreadmarksApi) Malloc(size int) (int, error) {
	panic("implement me")
}

func (t *TreadmarksApi) Free(addr, size int) error {
	panic("implement me")
}

func (t *TreadmarksApi) Barrier(id uint8){
	t.sendBarrierRequest(id)
}

func (t *TreadmarksApi) Lock(id uint8){
	lock := t.locks[id]
	lock.Lock()
	defer lock.Unlock()
	if lock.locked {
		panic("Tried ToTs lock a lock that was already locked.")
	} else if lock.held {
		lock.locked = true
	} else {
		if t.sendLockAcquireRequest(t.getManagerId(id), id) != nil {
			panic("Error when sending lock acquire request.")
		}
	}
}

func (t *TreadmarksApi) Release(id uint8){
	lock := t.locks[id]
	lock.Lock()
	defer lock.Unlock()
	if !lock.locked{
		panic ("Tried ToTs unlock a lock that was not locked.")
	} else if lock.nextTimestamp != nil {
		t.createInterval()
		t.sendLockAcquireResponse(lock.nextId, id, t.getMissingIntervals(lock.nextTimestamp))
		lock.nextId = 0
		lock.nextTimestamp = nil
		lock.held = false
	}
	lock.locked = false
}



//----------------------------------------------------------------//
//                        Other main functions                    //
//----------------------------------------------------------------//

func (t *TreadmarksApi) onFault(addr int, faultType byte, accessType string, value byte) {
	pageNr := int16(addr/t.memory.GetPageSize())
	page := t.pagearray[pageNr]
	if !page.hasCopy {
		t.sendCopyRequest(pageNr)
		<- t.channel
	}



}

func (t *TreadmarksApi) initializeLocks() {
	var i uint8
	for i = 0; i < uint8(len(t.locks)); i++{
		t.locks[i] = Lock{new(sync.Mutex), false, t.myId == t.getManagerId(i), 0, 0, nil}
	}
}

func (t *TreadmarksApi) initializeBarriers() {
	for i := range t.barriers {
		t.barriers[i] = make(chan uint8, 1)
	}
}

func (t *TreadmarksApi) createInterval() IntervalRecord{
	pages := make([]int16, 0, len(t.twinMap))
	for key := range t.twinMap{
		pages = append(pages, key)
	}
	timestamp := NewTimestamp(t.nrProcs)
	copy(timestamp, t.timestamp)
	interval := IntervalRecord{
		owner: t.myId,
		timestamp: timestamp,
		pages: pages,
	}
	t.procarray[t.myId] = append(t.procarray[t.myId], interval)
	t.timestamp.increment(t.myId)
	return interval
}

func (t *TreadmarksApi) createWriteNotice(pageNr int16) {
	record := t.procarray[t.myId][len(t.procarray[t.myId])-1]
	page := t.pagearray[pageNr]
	if t.twinMap[pageNr] != nil{
		wn := WritenoticeRecord{
			timestamp: record.timestamp,
			diff: nil,
		}
		page.writenotices[t.myId] = append(page.writenotices[t.myId], wn)
	}
}

func (t *TreadmarksApi) createDiff(pageNr int16){
	diff := make(map[int]byte)
	page := t.pagearray[pageNr]
	newData := t.memory.PrivilegedRead(int(pageNr)*t.memory.GetPageSize(), t.memory.GetPageSize())
	twin := t.twinMap[pageNr]
	for i := range twin{
		if twin[i] != newData[i]{
			diff[i] = newData[i]
		}
	}
	delete(t.twinMap, pageNr)
	t.memory.SetRights(t.memory.GetPageSize()*int(pageNr), 1)
	page.writenotices[t.myId][len(page.writenotices[t.myId])-1].diff = diff
}

func (t *TreadmarksApi) getMissingIntervals(ts Timestamp) []IntervalRecord {
	var procId uint8
	intervals := make([]IntervalRecord, 0, 100)
	for procId = 0; procId < t.nrProcs; procId++{
		intervals = append(intervals, t.getMissingIntervalsForProc(procId, ts)...)
	}
	return intervals
}

func (t *TreadmarksApi) getMissingIntervalsForProc(procId uint8, ts Timestamp) []IntervalRecord {
	array := t.procarray[procId]
	var i int
	for i = range array{
		if !ts.covers(array[i].timestamp){
			break
		}
	}
	return array[i:]
}

func (t *TreadmarksApi) hasMissingDiffs(pageNr int16) bool {
	panic("imeplement me")
}

func (t *TreadmarksApi) createDiffRequests(pageNr int16) []DiffRequest {
	var proc uint8
	targetProcs := make([]uint8, 0)
	for proc = 0; proc < t.nrProcs; proc++ {
		target := true
		ts := t.getHighestTimestamp(proc)
		newTargets := make([]uint8, 0)
		for i := range targetProcs{
			ots := t.getHighestTimestamp(targetProcs[i])
			if !ts.covers(ots) {
				newTargets = append(newTargets, targetProcs[i])
			}
			if ots.covers(ts) {
				target = false
			}
		}
		if target {
			newTargets = append(newTargets, proc)
			targetProcs = newTargets
		}
	}
	diffRequests := make([]DiffRequest, t.nrProcs)
	page := t.pagearray[pageNr]
	for proc = 0; proc < t.nrProcs; proc++ {
		i := len(page.writenotices[proc])-1
		last := page.writenotices[proc][i]
		var diffRequest DiffRequest
		if last.diff == nil {
			var lastTs Timestamp
			var firstTs Timestamp
			copy(lastTs, page.writenotices[proc][i].timestamp)
			copy(firstTs, lastTs)
			var wn WritenoticeRecord
			for i > 0{
				i--
				wn = page.writenotices[proc][i]
				copy(firstTs, wn.timestamp)
				if wn.diff != nil {
					break
				}
			}
			diffRequest = DiffRequest{PageNr: pageNr, FromTs: firstTs, ToTs: lastTs}
			for i := range targetProcs {
				proc := targetProcs[i]
				if t.getHighestTimestamp(proc).covers(diffRequest.ToTs) {
					if diffRequests[proc].FromTs != nil {
						diffRequests[proc].ToTs = diffRequests[proc].ToTs.merge(diffRequest.ToTs)
						diffRequests[proc].FromTs = diffRequests[proc].FromTs.min(diffRequest.FromTs)
					}
					break
				}
			}
		}
	}
	return diffRequests
}

func (t *TreadmarksApi) sendCopyRequest(pageNr int16) {
	page := t.pagearray[pageNr]
	var to uint8
	to = 0
	if len(page.copySet) > 0 {
		to = page.copySet[0]
	}
	req := CopyRequest{
		PageNr:pageNr,
	}
	t.encLock.Lock()
	t.enc.Encode(req)
	msg := make([]byte, t.encBuff.Len()+2)
	msg[0] = Uint8ToByte(to)
	msg[1] = byte(5)
	t.encBuff.Read(msg[2:])
	t.encLock.Unlock()
	t.out <- msg
}

func (t *TreadmarksApi) sendCopyResponse(to uint8, pageNr int16) {
	pageSize := t.memory.GetPageSize()
	addr := int(pageNr)*pageSize
	resp := CopyResponse{
		PageNr:       pageNr,
		Data: t.memory.PrivilegedRead(addr, pageSize),
	}
	t.encLock.Lock()
	t.enc.Encode(resp)
	msg := make([]byte, t.encBuff.Len()+2)
	msg[0] = Uint8ToByte(to)
	msg[1] = byte(6)
	t.encBuff.Read(msg[2:])
	t.encLock.Unlock()
	t.out <- msg
}

func (t *TreadmarksApi) sendDiffRequests(pageNr int16) {
	diffRequests := t.createDiffRequests(pageNr)
	for i, req := range diffRequests {
		t.encLock.Lock()
		t.enc.Encode(req)
		msg := make([]byte, t.encBuff.Len()+2)
		msg[0] = Uint8ToByte(uint8(i))
		msg[1] = byte(7)
		t.encBuff.Read(msg[2:])
		t.encLock.Unlock()
		t.out <- msg
	}
	for range diffRequests{
		<- t.channel
	}
}

func (t *TreadmarksApi) sendDiffResponse(to uint8, pageNr int16, writenotices []WritenoticeRecord) {
	resp := DiffResponse{
		PageNr:       pageNr,
		Writenotices: writenotices,
	}
	t.encLock.Lock()
	t.enc.Encode(resp)
	msg := make([]byte, t.encBuff.Len()+2)
	msg[0] = Uint8ToByte(to)
	msg[1] = byte(1)
	t.encBuff.Read(msg[2:])
	t.encLock.Unlock()
	t.out <- msg
}


func (t *TreadmarksApi) sendLockAcquireResponse(to, lockId uint8, intervals []IntervalRecord) error{
	resp := LockAcquireResponse{
		LockId:    lockId,
		Intervals: intervals,
	}
	t.encLock.Lock()
	t.enc.Encode(resp)
	msg := make([]byte, t.encBuff.Len()+2)
	msg[0] = Uint8ToByte(to)
	msg[1] = byte(1)
	t.encBuff.Read(msg[2:])
	t.encLock.Unlock()
	t.out <- msg
	return nil
}

func (t *TreadmarksApi) sendBarrierRequest(barrierId uint8) error {
	managerId := t.getManagerId(barrierId)
	req := BarrierRequest{
		BarrierId: barrierId,
		Intervals: t.getMissingIntervalsForProc(managerId, t.getHighestTimestamp(managerId)),
	}
	t.encLock.Lock()
	t.enc.Encode(req)
	msg := make([]byte, t.encBuff.Len()+2)
	msg[0] = Uint8ToByte(managerId)
	msg[1] = byte(3)
	t.encBuff.Read(msg[2:])
	t.encLock.Unlock()
	t.out <- msg
	<- t.channel
	return nil
}

func (t *TreadmarksApi) sendBarrierResponse(to uint8, intervals []IntervalRecord) error {
	resp := BarrierResponse{
		Intervals: intervals,
	}
	t.encLock.Lock()
	t.enc.Encode(resp)
	msg := make([]byte, t.encBuff.Len()+2)
	msg[0] = Uint8ToByte(to)
	msg[1] = byte(4)
	t.encBuff.Read(msg[2:])
	t.encLock.Unlock()
	t.out <- msg
	return nil
}

func (t *TreadmarksApi) sendLockAcquireRequest(to uint8, lockId uint8)  error {
	req := LockAcquireRequest{
		LockId:    lockId,
		Timestamp: t.timestamp,
	}
	t.encLock.Lock()
	t.enc.Encode(req)
	msg := make([]byte, t.encBuff.Len()+2)
	msg[0] = Uint8ToByte(t.getManagerId(lockId))
	msg[1] = byte(0)
	t.encBuff.Read(msg[2:])
	t.encLock.Unlock()
	t.out <- msg
	<- t.channel
	return nil
}

func (t *TreadmarksApi) insertIntervals(intervals []IntervalRecord) {
	for _, interval := range intervals{
		t.timestamp = t.timestamp.merge(interval.timestamp)
		t.procarray[interval.owner] = append(t.procarray[interval.owner], interval)
		for _, p := range interval.pages {
			page := t.pagearray[p]
			access := t.memory.GetRights(t.memory.GetPageSize()*int(p))
			if access == 2 && t.twinMap[p] != nil{
				t.createWriteNotice(p)
				t.createDiff(p)
			}
			if access != 0 {
				t.memory.SetRights(t.memory.GetPageSize()*int(p), 0)
			}
			wn := WritenoticeRecord{
				timestamp: interval.timestamp,
				diff: nil,
			}
			page.writenotices[interval.owner] = append(page.writenotices[interval.owner], wn)
		}
	}
}

func (t *TreadmarksApi) getManagerId(id uint8) uint8 {
	return 0
}

func (t *TreadmarksApi) forwardLockAcquireRequest(id uint8, req LockAcquireRequest) error {
	t.encLock.Lock()
	t.enc.Encode(req)
	msg := make([]byte, t.encBuff.Len()+2)
	msg[0] = Uint8ToByte(id)
	msg[1] = byte(0)
	t.encBuff.Read(msg[2:])
	t.encLock.Unlock()
	t.out <- msg
	return nil
}

func (t *TreadmarksApi) getHighestTimestamp(procId uint8) Timestamp {
	return t.procarray[procId][len(t.procarray[procId])-1].timestamp
}


/*
			Handling indcoming messages
 */

func (t *TreadmarksApi) handleIncoming() {
	var buf bytes.Buffer
	dec := gob.NewDecoder(&buf)
	for msg := range t.in{
		buf.Write(msg[2:])
		switch msg[1] {
		case 0: //Lock acquire request
			var req LockAcquireRequest
			dec.Decode(&req)
			req.From = uint8(msg[0])
			t.handleLockAcquireRequest(req)
		case 1: //Lock acquire response
			var resp LockAcquireResponse
			dec.Decode(&resp)
			t.handleLockAcquireResponse(resp)
		case 2: //Lock release request
			panic("implement me")
		case 3: //Barrier Request
			var req BarrierRequest
			dec.Decode(&req)
			req.From = uint8(msg[0])
			t.handleBarrierRequest(req)
		case 4: //Barrier response
			var resp BarrierResponse
			dec.Decode(&resp)
			t.handleBarrierResponse(resp)
		case 5: //Copy request
			var req CopyRequest
			dec.Decode(&req)
			req.From = uint8(msg[0])
			t.handleCopyRequest(req)
		case 6: //Copy response
			var resp CopyResponse
			dec.Decode(&resp)
			t.handleCopeResponse(resp)
		case 7: // Diff request
			var req DiffRequest
			dec.Decode(&req)
			req.From = uint8(msg[0])
			t.handleDiffRequest(req)
		case 8: //Diff response
			var resp DiffResponse
			dec.Decode(&resp)
			t.handleDiffResponse(resp)
		}

	}
}

func (t *TreadmarksApi) handleLockAcquireRequest(req LockAcquireRequest) {
	id := req.LockId
	t.locks[id].Lock()
	defer t.locks[id].Unlock()
	lock := t.locks[id]
	if lock.locked {
		lock.nextTimestamp = req.Timestamp
		lock.nextId = req.From
	} else if lock.held{
		t.createInterval()
		t.sendLockAcquireResponse(req.From, id, t.getMissingIntervals(req.Timestamp))
		lock.last = req.From
	} else {
		t.forwardLockAcquireRequest(lock.last, req)
		lock.last = req.From
	}
}

func (t *TreadmarksApi) handleLockAcquireResponse(resp LockAcquireResponse) {
	id := resp.LockId
	lock := t.locks[id]
	lock.Lock()
	defer lock.Unlock()
	t.insertIntervals(resp.Intervals)
	lock.locked = true
	lock.held = true
	t.channel <- true
}

func (t *TreadmarksApi) handleBarrierRequest(req BarrierRequest) error {
	t.insertIntervals(req.Intervals)
	n := <- t.barriers[req.BarrierId]
	n++
	var i uint8
	if n == t.nrProcs {
		for i = 0; i < t.nrProcs; i++ {
			t.sendBarrierResponse(i, t.getMissingIntervals(t.getHighestTimestamp(i)))
		}
		n = 0
	}
	t.barriers[req.BarrierId] <- n
	return nil
}

func (t *TreadmarksApi) handleBarrierResponse(resp BarrierResponse) error {
	t.insertIntervals(resp.Intervals)
	t.channel <- true
	return nil
}

func (t *TreadmarksApi) handleDiffRequest(req DiffRequest) {
	page := t.pagearray[req.PageNr]
	writenotices := make([]WritenoticeRecord, 0)
	var proc uint8
	for proc = 0; proc < t.nrProcs; proc++{
		i := len(page.writenotices[proc])
		var wn WritenoticeRecord
		for i > 0{
			i--
			wn = page.writenotices[proc][i]
			if req.ToTs.covers(wn.timestamp){
				break
			}
		}
		for i > 0 {
			i--
			wn = page.writenotices[proc][i]
			if req.FromTs.covers(wn.timestamp){
				break
			}
			writenotices = append(writenotices, wn)
		}
	}
	t.sendDiffResponse(req.From, req.PageNr, writenotices)
}

func (t *TreadmarksApi) handleDiffResponse(resp DiffResponse) {
	wnl := t.pagearray[resp.PageNr].writenotices
	var proc uint8
	proc = 0
	i := 0

	for proc < t.nrProcs && i < len(resp.Writenotices){
		wn := resp.Writenotices[i]
		for j := len(wnl[proc]); j < 0;{
			j--
			ts1 := wn.timestamp
			ts2 := wnl[proc][j].timestamp
			if ts2.covers(ts1) && !ts1.covers(ts2) {
				continue
			} else if ts1.covers(ts2) && !ts2.covers(ts1) {
				proc++
				break
			} else {
				wnl[proc][j].diff = wn.diff
			}
		}
	}
	t.channel <- true
}

func (t *TreadmarksApi) handleCopyRequest(req CopyRequest) {
	t.sendCopyResponse(req.From, req.PageNr)
	page := t.pagearray[req.PageNr]
	page.copySet = append(page.copySet, req.From)
}

func (t *TreadmarksApi) handleCopeResponse(resp CopyResponse) {
	t.memory.PrivilegedWrite(int(resp.PageNr)*t.memory.GetPageSize(), resp.Data)
	page := t.pagearray[resp.PageNr]
	page.hasCopy = true
	page.copySet = append(page.copySet, t.myId)
	t.channel <- true
}

//----------------------------------------------------------------//
//                           Help functions                       //
//----------------------------------------------------------------//


