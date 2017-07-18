package tests

import (
	"DSM-project/treadmarks"
	"testing"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t,0,  vc1.Compare(vc2))
	vc1.Increment(1)
	assert.Equal(t, 1, vc1.Compare(vc2))
	vc2.Increment(2)
	assert.Equal(t, 0, vc1.Compare(vc2))
}

func TestVectorMerge(t *testing.T){
	vc1 := treadmarks.NewVectorclock(4)
	vc2 := treadmarks.NewVectorclock(4)
	vc3 := treadmarks.NewVectorclock(4)
	vc4 := treadmarks.NewVectorclock(4)
	vc1.SetTick(1, 4)
	vcm := vc1.Merge(vc2)
	assert.Equal(t, uint(4), vcm.GetTick(1))
	vc3.SetTick(2, 5)
	vc4.SetTick(3, 4)
	vcm = vc3.Merge(vc4)
	assert.Equal(t, uint(5), vcm.GetTick(2))
	assert.Equal(t, uint(4), vcm.GetTick(3))
}