package treadmarks

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPageArray(t *testing.T) {
	//vm := memory.NewVmem(1024, 64)
	pageArray := NewPageArray(5)
	wn1 := WriteNoticeRecord{Diff: nil, Interval: nil}
	wn2 := WriteNoticeRecord{Diff: nil, Interval: nil}
	pe := NewPageArrayEntry(5)
	pe.copySet = []int{1, 2}
	pageArray.SetPageEntry(1, pe)

	pe.writeNoticeRecordArray[0] = []WriteNoticeRecord{wn1, wn2}

	assert.Equal(t, wn1, pageArray.GetWritenoticeList(0, 1)[0])
	assert.Equal(t, wn2, pageArray.GetWritenoticeList(0, 1)[1])
}

func TestPageArrayEntry_AddWriteNotice(t *testing.T) {
	pe := NewPageArrayEntry(1)
	wn1 := pe.PrependWriteNotice(byte(0), WriteNotice{})
	wn1.Id = 1
	assert.True(t, wn1 == pe.GetWriteNotice(0, 0))
	wn2 := pe.PrependWriteNotice(byte(0), WriteNotice{})
	wn2.Id = 2
	assert.True(t, wn2 == pe.GetWriteNotice(0, 0))
	assert.Equal(t, wn1, pe.GetWriteNotice(0, 1))
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
