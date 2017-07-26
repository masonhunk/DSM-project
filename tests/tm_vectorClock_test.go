package tests

import (
	"DSM-project/treadmarks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVectorclockIncrement(t *testing.T) {
	vc := treadmarks.NewVectorclock(4)
	assert.Equal(t, uint(0), vc.GetTick(0))
	vc.Increment(0)
	assert.Equal(t, uint(1), vc.GetTick(0))
}

func TestVectorclockCompare(t *testing.T) {
	vc1 := treadmarks.NewVectorclock(4)
	vc2 := treadmarks.NewVectorclock(4)
	assert.Equal(t, 0, vc1.Compare(*vc2))
	vc1.Increment(1)
	assert.Equal(t, 1, vc1.Compare(*vc2))
	vc2.Increment(2)
	assert.Equal(t, 0, vc1.Compare(*vc2))
}

func TestVectorMerge1(t *testing.T) {
	vc1 := treadmarks.NewVectorclock(4)
	vc2 := treadmarks.NewVectorclock(4)
	vc3 := treadmarks.NewVectorclock(4)
	vc4 := treadmarks.NewVectorclock(4)
	vc1.SetTick(1, 4)
	vcm := vc1.Merge(*vc2)
	assert.Equal(t, uint(4), vcm.GetTick(1))
	vc3.SetTick(2, 5)
	vc4.SetTick(3, 4)
	vcm = vc3.Merge(*vc4)
	assert.Equal(t, uint(5), vcm.GetTick(2))
	assert.Equal(t, uint(4), vcm.GetTick(3))
}

func TestVectorMerge2(t *testing.T) {
	vc1 := treadmarks.NewVectorclock(4)
	vc2 := treadmarks.NewVectorclock(4)
	vc3 := treadmarks.NewVectorclock(4)
	vc4 := treadmarks.NewVectorclock(4)
	assert.Equal(t, *vc1, treadmarks.Vectorclock{[]uint{0, 0, 0, 0}})
	assert.Equal(t, *vc2, treadmarks.Vectorclock{[]uint{0, 0, 0, 0}})
	assert.Equal(t, *vc3, treadmarks.Vectorclock{[]uint{0, 0, 0, 0}})
	assert.Equal(t, *vc4, treadmarks.Vectorclock{[]uint{0, 0, 0, 0}})
	vc1.SetTick(1, 4)
	assert.Equal(t, *vc1, treadmarks.Vectorclock{[]uint{4, 0, 0, 0}})
	vcm := vc1.Merge(*vc2)
	assert.Equal(t, *vcm, treadmarks.Vectorclock{[]uint{4, 0, 0, 0}})
	vc3.SetTick(2, 5)
	vc4.SetTick(3, 4)
	assert.Equal(t, *vc3, treadmarks.Vectorclock{[]uint{0, 5, 0, 0}})
	assert.Equal(t, *vc4, treadmarks.Vectorclock{[]uint{0, 0, 4, 0}})
	vcm = vc3.Merge(*vc4)
	assert.Equal(t, *vcm, treadmarks.Vectorclock{[]uint{0, 5, 4, 0}})
	assert.Equal(t, uint(5), vcm.GetTick(2))
	assert.Equal(t, uint(4), vcm.GetTick(3))
}

func TestVectorBefore(t *testing.T) {
	vc1 := treadmarks.NewVectorclock(3)
	vc2 := treadmarks.NewVectorclock(3)
	vc3 := treadmarks.NewVectorclock(3)
	vc2.SetTick(byte(1), 4)
	vc3.SetTick(byte(2), 4)
	assert.True(t, vc1.IsBefore(*vc2))
	assert.True(t, vc1.IsBefore(*vc3))
	assert.False(t, vc2.IsBefore(*vc1))
	assert.False(t, vc3.IsBefore(*vc1))
	assert.False(t, vc2.IsBefore(*vc3))
	assert.False(t, vc1.IsBefore(*vc1))
}
func TestVectorAfter(t *testing.T) {
	vc1 := treadmarks.NewVectorclock(3)
	vc2 := treadmarks.NewVectorclock(3)
	vc3 := treadmarks.NewVectorclock(3)
	vc2.SetTick(byte(1), 4)
	vc3.SetTick(byte(2), 4)
	assert.False(t, vc1.IsAfter(*vc2))
	assert.False(t, vc1.IsAfter(*vc3))
	assert.True(t, vc2.IsAfter(*vc1))
	assert.True(t, vc3.IsAfter(*vc1))
	assert.False(t, vc2.IsAfter(*vc3))
	assert.False(t, vc1.IsAfter(*vc1))
}

func TestVectorEqual(t *testing.T) {
	vc1 := treadmarks.NewVectorclock(3)
	vc2 := treadmarks.NewVectorclock(3)
	vc3 := treadmarks.NewVectorclock(3)
	vc3.SetTick(byte(0), 4)
	assert.True(t, vc1.Equals(*vc2))
	assert.False(t, vc1.Equals(*vc3))
}
