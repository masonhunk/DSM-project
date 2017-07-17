package tests

import (
	"DSM-project/treadmarks"
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
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
	fmt.Println(vc1)
	fmt.Println(vc2)
	assert.Equal(t, 0, vc1.Compare(vc2))
}