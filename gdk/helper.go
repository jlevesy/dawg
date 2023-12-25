package gdk

import (
	"encoding/json"
	"unsafe"
)

const (
	InputPath = "/dawg/input"
)

func WriteOutput(buf []byte) uint64 {
	bufPtr := &buf[0]
	unsafePtr := uintptr(unsafe.Pointer(bufPtr))

	ptr := uint32(unsafePtr)
	size := uint32(len(buf))

	return (uint64(ptr) << uint64(32)) | uint64(size)
}

type RuntimeError struct {
	Err string `json:"err"`
}

func Error(err error) uint64 {
	b, _ := json.Marshal(&RuntimeError{Err: err.Error()})
	return WriteOutput(append([]byte{'e'}, b...))
}
