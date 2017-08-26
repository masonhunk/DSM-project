package treadmarks

import (
	"testing"
	"github.com/stretchr/testify/assert"
)


func TestTimestamp_increment(t *testing.T) {
	ts1 := Timestamp{0, 0, 0}
	ts1.increment(1)
	assert.Equal(t, Timestamp{0, 1, 0}, ts1)
}

func TestTimestamp_cover(t *testing.T) {
	ts1 := Timestamp{0, 0, 0}
	ts2 := Timestamp{1, 0, 0}
	ts3 := Timestamp{0, 0, 1}
	assert.True(t, ts1.covers(ts1))
	assert.False(t, ts1.covers(ts2))
	assert.False(t, ts1.covers(ts3))
	assert.True(t, ts2.covers(ts1))
	assert.True(t, ts2.covers(ts2))
	assert.False(t, ts2.covers(ts3))
	assert.True(t, ts3.covers(ts1))
	assert.False(t, ts3.covers(ts2))
	assert.True(t, ts3.covers(ts3))
}

func TestTimestamp_loadsave(t *testing.T) {
	ts1 := Timestamp{0, 0, 0}
	data := ts1.saveToData()
	for i := range data{
		assert.Equal(t, byte(0), data[i])
	}
	ts1.increment(1)
	data = ts1.saveToData()
	ts2 := Timestamp{0, 0, 0}
	ts2.loadFromData(data)
	assert.Equal(t, ts1, ts2)

}
