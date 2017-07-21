package treadmarks

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPageArray(t *testing.T) {
	//vm := memory.NewVmem(1024, 64)
	pageArray := make(PageArray)
	wn1 := WriteNoticeRecord{Diff: nil, Interval: nil}
	wn2 := WriteNoticeRecord{Diff: nil, Interval: nil}
	pageArray[1] = &PageArrayEntry{
		CopySet:                []int{1, 2},
		WriteNoticeRecordArray: make(map[byte][]WriteNoticeRecord),
	}
	pageArray[1].WriteNoticeRecordArray[0] = []WriteNoticeRecord{wn1, wn2}
	assert.Equal(t, wn1, pageArray[1].WriteNoticeRecordArray[0][0])
	assert.Equal(t, wn2, pageArray[1].WriteNoticeRecordArray[0][1])

}

func TestPageArrayEntry_AddWriteNotice(t *testing.T) {
	pe := NewPageArrayEntry()
	wn1 := pe.PrependWriteNoticeOnPageArrayEntry(byte(0))
	wn1.id = 1
	assert.True(t, wn1 == &pe.WriteNoticeRecordArray[0][0])
	wn2 := pe.PrependWriteNoticeOnPageArrayEntry(byte(0))
	wn2.id = 2
	assert.True(t, wn2 == &pe.WriteNoticeRecordArray[0][0])
	assert.Equal(t, *wn1, pe.WriteNoticeRecordArray[0][1])
}

// This was just a proof of concept, to make sure pointers to writenoticerecords wouldnt break when
// the slice is extended.
func TestPageArrayEntry(t *testing.T) {
	s := []int{}
	for i := 0; i <= 8; i++ {
		s = append(s, i)
	}
	n := &s[4]
	fmt.Println(*n)
	*n = 6
	fmt.Println(s[4])
	fmt.Println(*n)
	s[4] = 2
	fmt.Println(s[4])
	fmt.Println(*n)
	for i := 9; i <= 10000; i++ {
		s = append([]int{i}, s...)
	}
	fmt.Println(*n)
}
