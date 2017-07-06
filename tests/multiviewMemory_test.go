package tests

import (
	"testing"
	"DSM-project/memory"
	"github.com/stretchr/testify/assert"
	"DSM-project/multiview"
)

func TestMultiViewMalloc(t *testing.T) {
	m := memory.NewVmem(4096, 128)
	mem := multiview.NewMVMem(m)

	ptr, err := mem.Malloc(100)
	assert.Nil(t, err)
	assert.Equal(t, 0, ptr)
}

