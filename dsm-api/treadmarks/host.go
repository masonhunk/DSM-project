package treadmarks

import (
	"DSM-project/dsm-api"
	"DSM-project/memory"
	"DSM-project/network"
	"DSM-project/utils"
	"bytes"
	"fmt"
	"github.com/davecgh/go-xdr/xdr2"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type TreadmarksApi struct {
	shutdown                       chan bool
	memory                         memory.VirtualMemory
	myId, nrProcs                  uint8
	memSize, pageByteSize, nrPages int
	pagearray                      []*pageArrayEntry
	twins                          [][]byte
	dirtyPages                     map[int16]bool
	dirtyPagesLock                 *sync.RWMutex
	procarray                      [][]IntervalRecord
	locks                          []*lock
	channel                        chan bool
	barrier                        chan uint8
	barrierreq                     []BarrierRequest
	in                             <-chan []byte
	out                            chan<- []byte
	conn                           network.Connection
	group                          *sync.WaitGroup
	timestamp                      Timestamp
}

var _ dsm_api.DSMApiInterface = new(TreadmarksApi)

func NewTreadmarksApi(memSize, pageByteSize int, nrProcs, nrLocks, nrBarriers uint8) (*TreadmarksApi, error) {
	var err error
	t := new(TreadmarksApi)
	t.memory = memory.NewVmem(memSize, pageByteSize)
	t.nrPages = memSize/pageByteSize + utils.Min(memSize%pageByteSize, 1)
	t.memSize, t.pageByteSize, t.nrProcs = memSize, pageByteSize, nrProcs
	t.pagearray = NewPageArray(t.nrPages, nrProcs)
	t.procarray = NewProcArray(nrProcs)
	t.locks = make([]*lock, nrLocks)
	t.channel = make(chan bool, 1)
	t.twins = make([][]byte, t.nrPages)
	t.dirtyPages = make(map[int16]bool)
	t.barrierreq = make([]BarrierRequest, t.nrProcs)
	t.timestamp = NewTimestamp(t.nrProcs)
	t.dirtyPagesLock = new(sync.RWMutex)
	return t, err
}

//----------------------------------------------------------------//
//              Functions defined by the interface                //
//----------------------------------------------------------------//

func (t *TreadmarksApi) Initialize(port int) error {
	t.conn, t.in, t.out = network.NewConnection(port, 10)
	t.memory.AddFaultListener(t.onFault)
	t.group = new(sync.WaitGroup)
	t.initializeBarriers()
	t.initializeLocks()
	t.shutdown = make(chan bool)
	go t.handleIncoming()
	return nil
}

func (t *TreadmarksApi) Join(ip string, port int) error {
	id, err := t.conn.Connect(ip, port)
	if err != nil {
		return err
	}
	t.myId = uint8(id)
	t.initializeLocks()
	t.timestamp = NewTimestamp(t.nrProcs)
	return nil
}

func (t *TreadmarksApi) Shutdown() error {
	t.shutdown <- true
	t.group.Wait()
	t.conn.Close()
	return nil
}

func (t *TreadmarksApi) Read(addr int) (byte, error) {
	b, err := t.memory.Read(addr)
	if err != nil {
		return t.memory.Read(addr)
	}
	return b, err
}

func (t *TreadmarksApi) Write(addr int, val byte) error {
	err := t.memory.Write(addr, val)
	if err != nil {
		return t.memory.Write(addr, val)
	}
	return nil
}

func (t *TreadmarksApi) Malloc(size int) (int, error) {
	return t.memory.Malloc(size)
}

func (t *TreadmarksApi) Free(addr, size int) error {
	return t.memory.Free(addr)
}

func (t *TreadmarksApi) Barrier(id uint8) {
	t.newInterval()

	t.sendBarrierRequest(id)
}

func (t *TreadmarksApi) AcquireLock(id uint8) {
	time.Sleep(0)
	lock := t.locks[id]
	lock.Lock()
	if lock.haveToken {
		if lock.locked {
			panic("locking lock twice")
		}
		lock.locked = true
		lock.Unlock()
	} else {
		lock.Unlock()
		t.newInterval()
		t.sendLockAcquireRequest(t.getManagerId(id), id)
	}
}

func (t *TreadmarksApi) ReleaseLock(id uint8) {
	lock := t.locks[id]
	lock.Lock()
	defer lock.Unlock()
	lock.locked = false
	if lock.nextTimestamp != nil {
		t.newInterval()
		t.sendLockAcquireResponse(id, lock.nextId, lock.nextTimestamp)
		lock.nextTimestamp = nil
		lock.nextId = t.getManagerId(id)
		lock.last = t.getManagerId(id)
		lock.haveToken = false
	}
}

//----------------------------------------------------------------//
//                        Initialisation                          //
//----------------------------------------------------------------//

func (t *TreadmarksApi) onFault(addr int, faultType byte, accessType string, value byte) {
	pageNr := int16(addr / t.memory.GetPageSize())
	page := t.pagearray[pageNr]
	access := t.memory.GetRights(addr)

	if access == memory.NO_ACCESS {
		if !page.hasCopy {
			t.sendCopyRequest(pageNr)
		}
		if t.hasMissingDiffs(pageNr) {
			t.sendDiffRequests(pageNr)
		}
	}
	if accessType == "WRITE" {
		t.twins[pageNr] = make([]byte, t.pageByteSize)
		copy(t.twins[pageNr], t.memory.PrivilegedRead(t.memory.GetPageAddr(int(pageNr)), t.memory.GetPageSize()))
		t.dirtyPagesLock.Lock()
		t.dirtyPages[pageNr] = true
		t.dirtyPagesLock.Unlock()
		t.memory.SetRights(addr, memory.READ_WRITE)
	} else {
		t.memory.SetRights(addr, memory.READ_ONLY)
	}
}

func (t *TreadmarksApi) initializeLocks() {
	var i uint8
	for i = 0; i < uint8(len(t.locks)); i++ {
		t.locks[i] = &lock{new(sync.Mutex), false, t.myId == t.getManagerId(i), t.getManagerId(i), 0, nil}
	}
}

func (t *TreadmarksApi) initializeBarriers() {
	t.barrier = make(chan uint8, 1)
	t.barrier <- 0
}

//----------------------------------------------------------------//
//                       Data Generation                          //
//----------------------------------------------------------------//

func (t *TreadmarksApi) newWritenoticeRecord(pageNr int16) {
	wn := WritenoticeRecord{
		Owner:     t.myId,
		Timestamp: t.timestamp,
	}
	page := t.pagearray[pageNr]
	fmt.Println(t.myId, t.timestamp, "Creating writenotices for page ", pageNr)
	fmt.Println(t.myId, t.timestamp, "Before adding writenotice we have the following list: ")
	fmt.Println(t.myId, t.timestamp, " -- ", page.writenotices[t.myId])
	page.writenotices[t.myId] = append(page.writenotices[t.myId], wn)
	fmt.Println(t.myId, t.timestamp, "After adding the writenotice, we have the following list of writenotices: ")
	fmt.Println(t.myId, t.timestamp, " -- ", t.pagearray[pageNr].writenotices[t.myId])
	delete(t.dirtyPages, pageNr)
}

func (t *TreadmarksApi) addWritenoticeRecord(pageNr int16, procId uint8, timestamp Timestamp) {
	pageSize := t.memory.GetPageSize()
	addr := int(pageNr) * pageSize
	access := t.memory.GetRights(addr)
	if access == memory.READ_WRITE {
		t.newWritenoticeRecord(pageNr)
		t.generateDiff(pageNr)
	}
	t.memory.SetRights(addr, memory.NO_ACCESS)
	wn := WritenoticeRecord{
		Owner:     procId,
		Timestamp: timestamp,
	}
	wnl := t.pagearray[pageNr].writenotices[procId]
	t.pagearray[pageNr].writenotices[procId] = append(wnl, wn)
	t.pagearray[pageNr].hasMissingDiffs = true
}

func (t *TreadmarksApi) newInterval() {

	t.dirtyPagesLock.Lock()
	pages := make([]int16, 0, len(t.dirtyPages))
	for page := range t.dirtyPages {
		pages = append(pages, page)
		delete(t.dirtyPages, page)
	}
	if len(pages) > 0 {

		t.timestamp = t.timestamp.increment(t.myId)
		fmt.Println(t.myId, t.timestamp, " Creating writenotice!")
		for _, page := range pages {
			t.newWritenoticeRecord(page)
		}
		fmt.Println()
		interval := IntervalRecord{
			Owner:     t.myId,
			Timestamp: t.timestamp,
			Pages:     pages,
		}
		t.procarray[t.myId] = append(t.procarray[t.myId], interval)
		fmt.Println(t.myId, t.timestamp, " Now my intervals are looking like this:")
		fmt.Println(t.myId, t.timestamp, t.procarray[t.myId])
	}

	t.dirtyPagesLock.Unlock()
}

func (t *TreadmarksApi) addInterval(interval IntervalRecord) {
	if !t.timestamp.covers(interval.Timestamp) {
		for _, p := range interval.Pages {
			t.addWritenoticeRecord(p, interval.Owner, interval.Timestamp)
		}
		t.procarray[interval.Owner] = append(t.procarray[interval.Owner], interval)
		t.timestamp = t.timestamp.merge(interval.Timestamp)
	}
}

func (t *TreadmarksApi) generateDiff(pageNr int16) {
	pageSize := t.memory.GetPageSize()
	addr := int(pageNr) * pageSize
	t.memory.SetRights(addr, memory.READ_ONLY)
	data := t.memory.PrivilegedRead(addr, pageSize)
	twin := t.twins[pageNr]
	diff := make(map[int]byte)
	for i := range data {
		if data[i] != twin[i] {
			diff[i] = data[i]
		}
	}
	t.twins[pageNr] = nil
	t.pagearray[pageNr].writenotices[t.myId][len(t.pagearray[pageNr].writenotices[t.myId])-1].Diff = diff

}

//----------------------------------------------------------------//
//                           Data Query                           //
//----------------------------------------------------------------//

func (t *TreadmarksApi) getMissingIntervals(ts Timestamp) []IntervalRecord {
	var proc uint8
	intervals := make([]IntervalRecord, 0, int(t.nrProcs)*5)
	for proc = 0; proc < t.nrProcs; proc++ {
		intervals = append(intervals, t.getMissingIntervalsForProc(proc, ts)...)
	}
	return intervals
}

func (t *TreadmarksApi) getMissingIntervalsForProc(procId uint8, ts Timestamp) []IntervalRecord {
	intervals := t.procarray[procId]
	result := make([]IntervalRecord, 0, len(intervals))
	for i := len(intervals) - 1; i >= 0; i-- {
		if !ts.covers(intervals[i].Timestamp) {
			result = append(result, intervals[i])
		} else {
			break
		}
	}
	return result
}

func (t *TreadmarksApi) hasMissingDiffs(pageNr int16) bool {
	return t.pagearray[pageNr].hasMissingDiffs
}

func (t *TreadmarksApi) createDiffRequests(pageNr int16) []DiffRequest {
	fmt.Println(t.myId, t.timestamp, " Creating diff request for page ", pageNr)
	diffRequests := make([]DiffRequest, 0, t.nrProcs)
	var proc uint8
	for proc = 0; proc < t.nrProcs; proc++ {
		req := t.createDiffRequest(pageNr, proc)
		if req.Last != nil {
			insert := true
			for i, oReq := range diffRequests {
				if oReq.Last.covers(req.Last) {
					diffRequests[i].First = oReq.First.min(req.First)
					insert = false
				} else if req.Last.covers(oReq.Last) {
					diffRequests[i].to = req.to
					diffRequests[i].First = oReq.First.min(req.First)
					insert = false
				}
			}
			if insert {
				diffRequests = append(diffRequests, req)
			}
		}

	}
	return diffRequests
}

func (t *TreadmarksApi) createDiffRequest(pageNr int16, procId uint8) DiffRequest {

	req := DiffRequest{
		to:     procId,
		From:   t.myId,
		PageNr: pageNr,
	}

	wnl := t.pagearray[pageNr].writenotices[procId]
	fmt.Println(t.myId, t.timestamp, " For proc ", procId, " i have the following writenotices:")
	for _, w := range wnl {
		fmt.Println(t.myId, "|-- Owner:", w.Owner, ", ts: ", w.Timestamp, " diff?:", w.Diff != nil)
	}
	l := len(wnl) - 1
	if l >= 0 && wnl[l].Diff == nil {
		req.Last = wnl[l].Timestamp
		for ; l >= 0; l-- {
			if wnl[l].Diff != nil {
				break
			}
			req.First = wnl[l].Timestamp
		}
	}
	fmt.Println(t.myId, t.timestamp, " The missing diffs are in the interval ", req)
	return req
}

//----------------------------------------------------------------//
//                         Send messages                          //
//----------------------------------------------------------------//

func (t *TreadmarksApi) sendMessage(to, msgType uint8, msg interface{}) {
	fmt.Println(t.myId, t.timestamp, " sending ", reflect.TypeOf(msg), " to ", to, " : ", msg)
	var w bytes.Buffer
	xdr.Marshal(&w, &msg)
	data := make([]byte, w.Len()+2)
	data[0] = byte(to)
	data[1] = byte(msgType)
	w.Read(data[2:])
	t.out <- data
}

func (t *TreadmarksApi) sendLockAcquireRequest(to uint8, lockId uint8) {
	req := LockAcquireRequest{
		From:      t.myId,
		LockId:    lockId,
		Timestamp: t.timestamp,
	}

	t.sendMessage(to, 0, req)
	<-t.channel
}

func (t *TreadmarksApi) sendLockAcquireResponse(lockId uint8, to uint8, timestamp Timestamp) {
	intervals := t.getMissingIntervals(timestamp)
	resp := LockAcquireResponse{
		LockId:    lockId,
		Intervals: intervals,
		Timestamp: t.timestamp,
	}
	t.sendMessage(to, 1, resp)
}

func (t *TreadmarksApi) forwardLockAcquireRequest(id uint8, req LockAcquireRequest) {
	t.sendMessage(id, 0, req)
}

func (t *TreadmarksApi) sendBarrierRequest(barrierId uint8) {
	managerId := t.getManagerId(barrierId)
	req := BarrierRequest{
		From:      t.myId,
		BarrierId: barrierId,
		Timestamp: t.timestamp,
	}
	if t.myId != managerId {
		req.Intervals = t.getMissingIntervalsForProc(t.myId, t.getHighestTimestamp(managerId))
		t.sendMessage(managerId, 3, req)
	} else {
		t.handleBarrierRequest(req)
	}
	<-t.channel
}

func (t *TreadmarksApi) sendBarrierResponse(to uint8, ts Timestamp) {
	resp := BarrierResponse{
		Intervals: t.getMissingIntervals(ts),
		Timestamp: t.timestamp,
	}
	t.sendMessage(to, 4, resp)
}

func (t *TreadmarksApi) sendCopyRequest(pageNr int16) {
	page := t.pagearray[pageNr]
	copySet := page.copySet
	to := copySet[len(copySet)-1]
	if to != t.myId {
		req := CopyRequest{
			From:   t.myId,
			PageNr: pageNr,
		}
		t.sendMessage(to, 5, req)
	} else {
		page.hasCopy = true
		t.channel <- true
	}
	<-t.channel
}

func (t *TreadmarksApi) sendCopyResponse(to uint8, pageNr int16) {
	data := make([]byte, t.pageByteSize)
	copy(data, t.twins[pageNr])
	if data == nil {
		pageSize := t.memory.GetPageSize()
		addr := int(pageNr) * pageSize
		copy(data, t.memory.PrivilegedRead(addr, pageSize))
	}
	resp := CopyResponse{
		PageNr: pageNr,
		Data:   data,
	}
	t.sendMessage(to, 6, resp)
}

func (t *TreadmarksApi) sendDiffRequests(pageNr int16) {
	diffRequests := t.createDiffRequests(pageNr)
	for _, req := range diffRequests {
		t.sendMessage(req.to, 7, req)
	}
	for range diffRequests {
		<-t.channel
	}
	t.applyAllDiffs(pageNr)
}

func (t *TreadmarksApi) sendDiffResponse(to uint8, pageNr int16, writenotices []WritenoticeRecord) {
	resp := DiffResponse{
		PageNr:       pageNr,
		Writenotices: writenotices,
	}
	t.sendMessage(to, 8, resp)
}

func (t *TreadmarksApi) getManagerId(id uint8) uint8 {
	return 0
}

func (t *TreadmarksApi) getHighestTimestamp(procId uint8) Timestamp {
	if len(t.procarray[procId]) == 0 {
		return NewTimestamp(t.nrProcs)
	}
	ts := t.procarray[procId][len(t.procarray[procId])-1].Timestamp
	return ts
}

//----------------------------------------------------------------//
//                  Handling incoming messages                    //
//----------------------------------------------------------------//

func (t *TreadmarksApi) handleIncoming() {
	t.group.Add(1)
	buf := bytes.NewBuffer([]byte{})
Loop:
	for {
		var msg []byte
		select {
		case msg = <-t.in:
		case <-t.shutdown:
			break Loop
		}
		buf.Write(msg[2:])
		switch msg[1] {
		case 0: //lock acquire request
			var req LockAcquireRequest
			_, err := xdr.Unmarshal(buf, &req)
			if err != nil {
				panic(err.Error())
			}
			t.handleLockAcquireRequest(req)
		case 1: //lock acquire response
			var resp LockAcquireResponse
			_, err := xdr.Unmarshal(buf, &resp)
			if err != nil {
				panic(err.Error())
			}
			t.handleLockAcquireResponse(resp)
		case 3: //Barrier Request
			var req BarrierRequest
			_, err := xdr.Unmarshal(buf, &req)
			if err != nil {
				panic(err.Error())
			}
			t.handleBarrierRequest(req)
		case 4: //Barrier response
			var resp BarrierResponse
			_, err := xdr.Unmarshal(buf, &resp)
			if err != nil {
				panic(err.Error())
			}
			t.handleBarrierResponse(resp)
		case 5: //Copy request
			var req CopyRequest
			_, err := xdr.Unmarshal(buf, &req)
			if err != nil {
				panic(err.Error())
			}
			t.handleCopyRequest(req)
		case 6: //Copy response
			var resp CopyResponse
			_, err := xdr.Unmarshal(buf, &resp)
			if err != nil {
				panic(err.Error())
			}
			t.handleCopyResponse(resp)
		case 7: // Diff request
			var req DiffRequest
			_, err := xdr.Unmarshal(buf, &req)
			if err != nil {
				panic(err.Error())
			}
			req.From = uint8(msg[0])
			t.handleDiffRequest(req)
		case 8: //Diff response
			var resp DiffResponse
			_, err := xdr.Unmarshal(buf, &resp)
			if err != nil {
				panic(err.Error())
			}
			t.handleDiffResponse(resp)
		}
	}
	t.group.Done()
}

func (t *TreadmarksApi) handleLockAcquireRequest(req LockAcquireRequest) {
	id := req.LockId
	lock := t.locks[id]
	lock.Lock()
	if !lock.haveToken || lock.locked {
		t.addToLockQueue(req)
	} else {
		t.newInterval()
		t.sendLockAcquireResponse(id, req.From, req.Timestamp)
		lock.haveToken = false
	}
	lock.last = req.From
	lock.Unlock()
}

func (t *TreadmarksApi) handleLockAcquireResponse(resp LockAcquireResponse) {
	id := resp.LockId
	lock := t.locks[id]
	lock.Lock()
	for i := len(resp.Intervals); i > 0; i-- {
		t.addInterval(resp.Intervals[i-1])
	}
	lock.locked = true
	lock.haveToken = true
	lock.Unlock()
	t.channel <- true
}

func (t *TreadmarksApi) handleBarrierRequest(req BarrierRequest) {
	t.barrierreq[req.From] = req
	n := <-t.barrier + 1
	if n < t.nrProcs {
		t.barrier <- n
	} else {
		for _, req := range t.barrierreq {
			for i := len(req.Intervals); i > 0; i-- {
				t.addInterval(req.Intervals[i-1])
			}
		}
		var i uint8
		for i = 0; i < t.nrProcs; i++ {
			if i != t.myId {
				t.sendBarrierResponse(i, t.barrierreq[i].Timestamp)
			}
		}
		t.barrier <- 0
		t.channel <- true
	}
}

func (t *TreadmarksApi) handleBarrierResponse(resp BarrierResponse) {
	for i := len(resp.Intervals); i > 0; i-- {
		t.addInterval(resp.Intervals[i-1])
	}

	t.channel <- true
}

func (t *TreadmarksApi) handleCopyRequest(req CopyRequest) {
	t.sendCopyResponse(req.From, req.PageNr)
	copyset := t.pagearray[req.PageNr].copySet
	copyset = append(copyset, req.From)
}

func (t *TreadmarksApi) handleCopyResponse(resp CopyResponse) {
	t.memory.PrivilegedWrite(int(resp.PageNr)*t.memory.GetPageSize(), resp.Data)
	page := t.pagearray[resp.PageNr]
	page.hasCopy = true
	page.copySet = append(page.copySet, t.myId)
	t.channel <- true
}

func (t *TreadmarksApi) handleDiffRequest(req DiffRequest) {
	fmt.Println(t.myId, t.timestamp, " got diff request: ", req)

	if t.twins[req.PageNr] != nil {
		t.generateDiff(req.PageNr)
	}
	fmt.Println(t.myId, t.timestamp, "For that page, I have the following writenotices:")
	var i uint8
	for i = 0; i < t.nrProcs; i++ {
		fmt.Println(t.myId, " --- proc: ", i, " - ", t.pagearray[req.PageNr].writenotices[i])
	}
	result := make([]WritenoticeRecord, 0)
	var proc uint8
	for proc = 0; proc < t.nrProcs; proc++ {
		if proc == req.From {
			continue
		}

		list := t.pagearray[req.PageNr].writenotices[proc]
		for i := len(list) - 1; i >= 0; i-- {
			wn := list[i]
			if !wn.Timestamp.covers(req.First) {
				break
			} else if req.Last.covers(wn.Timestamp) {
				result = append(result, wn)
			}
		}
	}
	fmt.Println(t.myId, t.timestamp, "sending following diffs as a response: ", result)
	t.sendDiffResponse(req.From, req.PageNr, result)
}

func (t *TreadmarksApi) handleDiffResponse(resp DiffResponse) {
	var proc uint8
	wnl := resp.Writenotices
	j := 0
	for proc = 0; proc < t.nrProcs; proc++ {
		if proc == t.myId {
			continue
		}
		if j >= len(wnl) {
			break
		}
		list := t.pagearray[resp.PageNr].writenotices[proc]

		for i := len(list) - 1; i >= 0; i-- {
			if !(j < len(wnl)) {
				break
			}
			if !list[i].Timestamp.equals(wnl[j].Timestamp) {
				break
			}
			list[i].Diff = wnl[j].Diff

			j++

		}
	}
	t.channel <- true
}

func (t *TreadmarksApi) applyAllDiffs(pageNr int16) {
	wnl := t.pagearray[pageNr].writenotices
	index := t.pagearray[pageNr].index
	for {
		var best uint8 = 0
		var bestTs Timestamp = nil
		var proc uint8
		for proc = 0; proc < t.nrProcs; proc++ {
			if len(wnl[proc]) > index[proc] {
				wn := wnl[proc][index[proc]]
				if bestTs == nil || !wn.Timestamp.covers(bestTs) {
					best = proc
					bestTs = wn.Timestamp
				}
			}
		}
		if bestTs == nil {
			break
		}
		diff := wnl[best][index[best]].Diff
		t.applyDiff(pageNr, diff)
		index[best]++
	}

}

func (t *TreadmarksApi) applyDiff(pageNr int16, diff map[int]byte) {
	size := t.memory.GetPageSize()
	addr := int(pageNr) * size
	data := t.memory.PrivilegedRead(addr, size)
	for key, value := range diff {
		data[key] = value
	}
	t.memory.PrivilegedWrite(addr, data)
}

func (t *TreadmarksApi) addToLockQueue(req LockAcquireRequest) {
	lockId := req.LockId
	lock := t.locks[lockId]
	if lock.nextTimestamp == nil && (t.myId == lock.last || t.myId != t.getManagerId(lockId)) {
		lock.nextTimestamp = req.Timestamp
		lock.nextId = req.From
	} else {
		t.forwardLockAcquireRequest(lock.last, req)
	}
}

//----------------------------------------------------------------//
//                           Help functions                       //
//----------------------------------------------------------------//

func (t *TreadmarksApi) GetId() int {
	return int(t.myId)
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
