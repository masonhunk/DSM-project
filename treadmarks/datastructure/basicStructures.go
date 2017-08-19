package datastructure

import (
	"DSM-project/treadmarks"
	"fmt"
)

type IntervalRecordInterface interface {
	GetId() string
	GetTimestamp() treadmarks.Timestamp
	GetWriteNoticeIds() []string
	AddWritenoticeId(wnId string)
}

type IntervalRecord struct{
	procId byte
	writeNoticeIds []string
	timestamp treadmarks.Timestamp
}

func NewIntervalRecord(procId byte, timestamp treadmarks.Timestamp) *IntervalRecord{
	i := new(IntervalRecord)
	i.procId = procId
	i.timestamp = timestamp
	i.writeNoticeIds = make([]string, 0)
	return i
}

func (i *IntervalRecord) GetId() string{
	return fmt.Sprint(i.procId, i.timestamp)
}

func (i *IntervalRecord) GetTimestamp() treadmarks.Timestamp {
	return i.timestamp
}

func (i *IntervalRecord) GetWriteNoticeIds() []string {
	return i.writeNoticeIds
}

func (i *IntervalRecord) AddWritenoticeId(wnId string) {
	i.writeNoticeIds = append(i.writeNoticeIds, wnId)
}

type WriteNoticeRecordInterface interface{
	GetIntervalId() string
	GetDiff() Diff
	SetDiff(diffs Diff)
	GetTimestamp() treadmarks.Timestamp
	GetId() string
}

type WriteNoticeRecord struct{
	id string
	intervalId string
	timestamp  treadmarks.Timestamp
	diffs      Diff
}

//Compile test to make sure WriteNoticeRecord implements the interface
var _ WriteNoticeRecordInterface = new(WriteNoticeRecord)

func NewWriteNoticeRecord(pageNr int, procId byte, timestamp treadmarks.Timestamp, diffs Diff) *WriteNoticeRecord{
	wn := WriteNoticeRecord{
		WriteNoticeId(pageNr, procId, timestamp),
		fmt.Sprint("int_", procId, timestamp),
		timestamp, diffs}
	return &wn
}

func WriteNoticeId(pageNr int, procId byte, timestamp treadmarks.Timestamp) string {
	return fmt.Sprint("wn_", pageNr, procId, timestamp)
}

func (w *WriteNoticeRecord) GetIntervalId() string {
	return w.intervalId
}

func (w *WriteNoticeRecord) GetTimestamp() treadmarks.Timestamp{
	return w.timestamp
}

func (w *WriteNoticeRecord) GetDiff() Diff {
	return w.diffs
}

func (w *WriteNoticeRecord) SetDiff(diffs Diff) {
	w.diffs = diffs
}

func (w *WriteNoticeRecord) GetId() string{
	return w.id
}

type ByVal []WriteNoticeRecordInterface

func (a ByVal) Len() int           { return len(a) }
func (a ByVal) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByVal) Less(i, j int) bool { return a[i].GetTimestamp().Less(a[j].GetTimestamp()) }


type Diff map[int]byte

type DiffDescription struct{
	ProcId byte
	Timestamp treadmarks.Timestamp
	Diff Diff
}

type DiffSort []DiffDescription

func (a DiffSort) Len() int           { return len(a) }
func (a DiffSort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DiffSort) Less(i, j int) bool { return a[i].Timestamp.Less(a[j].Timestamp) }