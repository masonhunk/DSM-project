package treadmarks

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPageArray(t *testing.T) {
	//vm := memory.NewVmem(1024, 64)
	pageArray := make(PageArray)
	wn1 := WriteNoticeRecord{Diff: nil, Interval: nil, NextRecord: nil, PrevRecord: nil}
	wn2 := WriteNoticeRecord{Diff: nil, Interval: nil, NextRecord: nil, PrevRecord: &wn1}
	wn1.NextRecord = &wn2
	pageArray[1] = PageArrayEntry{
		CopySet: []int{1, 2},
		ProcArr: make(map[byte]*WriteNoticeRecord),
	}
	pageArray[1].ProcArr[0] = &wn1
	assert.Equal(t, wn1, *pageArray[1].ProcArr[0])
	assert.Nil(t, pageArray[1].ProcArr[0].PrevRecord)
	assert.Equal(t, wn2, *pageArray[1].ProcArr[0].NextRecord)
	assert.Nil(t, pageArray[1].ProcArr[0].NextRecord.NextRecord)

}
