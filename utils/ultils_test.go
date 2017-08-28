package utils

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestUint8ToByteToUint8(t *testing.T) {
	var i uint8
	for i = 0; i < 255; i++ {
		assert.Equal(t, i, ByteToUint8(Uint8ToByte(i)))
	}
}

func TestInt16ToByteToInt16(t *testing.T) {
	var i int16
	for i = 0; i < 32767; i++ {
		assert.Equal(t, i, BytesToInt16(Int16ToBytes(i)))
	}
}

func TestInt32ToByteToInt32(t *testing.T) {
	var i int32
	for i = 0; i < 42767; i++ {
		assert.Equal(t, i, BytesToInt32(Int32ToBytes(i)))
	}
}