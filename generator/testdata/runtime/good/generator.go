package main

import (
	"github.com/jlevesy/dawg/gdk"
)

//export generate
func generate(ptr, size uint32) uint64 {
	return gdk.WriteOutput([]byte(`{"some":"config"}`))
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
