package treadmarks

import "unsafe"

func Min(x, y int) int {
	if x < y{
		return x
	}
	return y
}

func ByteToUint8(data byte) uint8{
	return *(*uint8)(unsafe.Pointer(&data))
}

func Uint8ToByte(i uint8) byte{
	return  *(*byte)(unsafe.Pointer(&i))
}

func BytesToInt16(data []byte) int16{
	return *(*int16)(unsafe.Pointer(&data[0]))
}

func Int16ToBytes(i int16) []byte{
	ptr := uintptr(unsafe.Pointer(&i))
	slice := make([]byte, 2)
	for i := 0; i < 2; i++ {
		slice[i] = *(*byte)(unsafe.Pointer(ptr))
		ptr++
	}
	return slice
}

func BytesToInt32(data []byte) int32{
	return *(*int32)(unsafe.Pointer(&data[0]))
}

func Int32ToBytes(i int32) []byte{
	ptr := uintptr(unsafe.Pointer(&i))
	slice := make([]byte, 4)
	for i := 0; i < 4; i++ {
		slice[i] = *(*byte)(unsafe.Pointer(ptr))
		ptr++
	}
	return slice
}