package treadmarks

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"sort"
)

func TestTimestamp_Less(t *testing.T) {
	//Testing with two equal timestamps, where the first shouldnt be less.
	var ts1 Timestamp = []int{0, 0, 0}
	var ts2 Timestamp = []int{0, 0, 0}
	assert.Equal(t, false, ts1.Less(ts2))

	//Testing where the first timestamp is the largest. Should be false.
	ts1 = []int{1, 0, 0}
	ts2 = []int{0, 0, 0}
	assert.Equal(t, false, ts1.Less(ts2))

	//Testing where the second timestamp is the largest. Should be true.
	ts1 = []int{0, 0, 0}
	ts2 = []int{1, 0, 0}
	assert.Equal(t, true, ts1.Less(ts2))

	//Testing where they are concurrent, but the first has the first largest tick. Should be false.
	ts1 = []int{1, 0, 0}
	ts2 = []int{0, 1, 0}
	assert.Equal(t, false, ts1.Less(ts2))

	//Testing where they are concurrent, but the second has the first largest tick. Should be true.
	ts1 = []int{0, 0, 1}
	ts2 = []int{0, 1, 0}
	assert.Equal(t, true, ts1.Less(ts2))

	//Just to ensure proper behavior, we have some edge cases:
	ts1 = []int{1, 0, 1}
	ts2 = []int{0, 1, 0}
	assert.Equal(t, false, ts1.Less(ts2))

	ts1 = []int{0, 1, 0}
	ts2 = []int{1, 0, 1}
	assert.Equal(t, true, ts1.Less(ts2))
}

func TestTimestamp_Sort(t *testing.T){
	//First a test where the list has a proper order.
	tsList := []Timestamp{
		[]int{0, 1, 2},
		[]int{0, 1, 0},
		[]int{1, 1, 2},
	}

	expected := []Timestamp{
		[]int{0, 1, 0},
		[]int{0, 1, 2},
		[]int{1, 1, 2},
	}
	sort.Sort(ByVal(tsList))
	assert.Equal(t, expected, tsList)

	//Then we try with something that doesn't have a proper order.
	tsList = []Timestamp{
		[]int{1, 0, 2},
		[]int{0, 1, 0},
		[]int{1, 1, 2},
	}

	expected = []Timestamp{
		[]int{0, 1, 0},
		[]int{1, 0, 2},
		[]int{1, 1, 2},
	}
	sort.Sort(ByVal(tsList))
	assert.Equal(t, expected, tsList)

	//Another test with improper ordering.
	tsList = []Timestamp{
		[]int{0, 2, 0},
		[]int{0, 0, 2},
		[]int{2, 0, 0},
	}

	expected = []Timestamp{
		[]int{0, 0, 2},
		[]int{0, 2, 0},
		[]int{2, 0, 0},
	}

	sort.Sort(ByVal(tsList))
	assert.Equal(t, expected, tsList)
}

func TestTimestamp_String(t *testing.T) {
	ts := NewTimestamp(2)
	assert.Equal(t, "[0,0]", ts.String())
	ts = ts.Increment(byte(1))
	assert.Equal(t, "[1,0]", ts.String())
	ts = ts.Increment(byte(2))
	assert.Equal(t, "[1,1]", ts.String())
	assert.Panics(t, func(){ts.Increment(byte(0))})
	assert.Panics(t, func(){ts.Increment(byte(3))})
}

func TestTimestamp_Equals(t *testing.T) {
	ts1 := NewTimestamp(2)
	ts2 := NewTimestamp(2)
	ts3 := NewTimestamp(3)
	assert.True(t, ts1.Equals(ts2))
	assert.False(t, ts1.Equals(ts3))
	ts1 = ts1.Increment(byte(1))
	assert.False(t, ts1.Equals(ts2))
	ts2 = ts2.Increment(byte(1))
	assert.True(t, ts1.Equals(ts2))
}
