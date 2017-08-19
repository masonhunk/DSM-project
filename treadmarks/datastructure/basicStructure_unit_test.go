package datastructure

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"DSM-project/treadmarks"
	"fmt"
)

func TestIntervalRecord(t *testing.T) {
	ts := treadmarks.NewTimestamp(3)
	i := NewIntervalRecord(byte(1), ts)
	assert.Equal(t, []string{}, i.GetWriteNoticeIds())
	assert.Equal(t, ts, i.GetTimestamp(), ts)
	assert.Equal(t, byte(1), i.procId)
	assert.Equal(t, fmt.Sprint(byte(1), ts), i.GetId())
	i.AddWritenoticeId("stuff")
	assert.Len(t, i.GetWriteNoticeIds(), 1)
	assert.Equal(t, "stuff", i.GetWriteNoticeIds()[0])
}

func TestWriteNoticeRecord(t *testing.T) {
	ts := treadmarks.NewTimestamp(3)
	w := NewWriteNoticeRecord(1, byte(1), ts, nil)
	assert.Equal(t, WriteNoticeId(1, byte(1), ts), w.GetId())
	assert.Equal(t, fmt.Sprint("int_", byte(1), ts), w.GetIntervalId())
	assert.Equal(t, ts, w.GetTimestamp())
	assert.Nil(t, w.GetDiff())
	diffs := make(map[int]byte)
	diffs[1] = byte(2)
	w.SetDiff(diffs)
	assert.Len(t, w.GetDiff(), 1)
	assert.Equal(t, byte(2), w.GetDiff()[1])
}