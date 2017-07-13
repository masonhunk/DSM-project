package multiview

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestInitialize(t *testing.T) {
	Initialize(4096, 128)
	ptr, err := mem.Malloc(1000)
	assert.Nil(t, err)
	fmt.Println(ptr)
}
