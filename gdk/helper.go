package gdk

import "unsafe"

func ReadInput(ptr, size uint32) []byte {
	return []byte(unsafe.String((*byte)(unsafe.Pointer(uintptr(ptr))), size))
}

func WriteOutput(buf []byte) uint64 {
	bufPtr := &buf[0]
	unsafePtr := uintptr(unsafe.Pointer(bufPtr))

	ptr := uint32(unsafePtr)
	size := uint32(len(buf))

	return (uint64(ptr) << uint64(32)) | uint64(size)
}
