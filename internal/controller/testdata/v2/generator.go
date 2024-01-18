package main

import (
	"github.com/jlevesy/dawg/gdk"
)

//export generate
func generate() uint64 {
	return gdk.WriteOutput([]byte(`{"version":"v2"}`))
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
