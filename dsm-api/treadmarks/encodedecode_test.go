package treadmarks

import (
	"bytes"
	"encoding/gob"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	intervals := make([]IntervalRecord, 1)
	intervals[0] = IntervalRecord{
		Owner:     2,
		Timestamp: NewTimestamp(3),
		Pages:     []int16{0, 1},
	}
	resp := BarrierResponse{
		Intervals: intervals,
	}

	respc := BarrierResponse{
		Intervals: make([]IntervalRecord, 0),
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(resp)
	assert.Nil(t, err)
	err = dec.Decode(&respc)
	assert.Nil(t, err)
	assert.Equal(t, resp, respc)
}

/*
type BarrierResponse struct{
	Intervals []IntervalRecord
}

type IntervalRecord struct{
	Owner     uint8
	timestamp timestamp
	Pages     []int16
}
*/
