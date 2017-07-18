package treadmarks

import (
	"DSM-project/memory"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateDiff(t *testing.T) {
	original := []byte{0, 1, 2, 3, 4}
	new := []byte{0, 1, 3, 5, 4}
	assert.Equal(t, []Pair{Pair{2, byte(3)}, Pair{3, byte(5)}}, CreateDiff(original, new).Diffs)
}

func TestUpdateDatastructures(t *testing.T) {
	vm := memory.NewVmem(128, 8)
	tm := NewTreadMarks(vm, 1, 1, 1)
	tm.procId = 3
	tm.copyMap[0] = []byte{4, 4, 4, 4, 4, 4, 4, 4}
	tm.copyMap[1] = []byte{1, 1, 1, 1, 1, 1, 1, 1}
	tm.copyMap[2] = []byte{2, 2, 2, 2, 2, 2, 2, 2}
	tm.updateDatastructures()
	fmt.Println()
}
