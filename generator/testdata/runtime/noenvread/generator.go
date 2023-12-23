package main

import (
	"os"

	"github.com/jlevesy/dawg/gdk"
)

//export generate
func generate() uint64 {
	return gdk.WriteOutput([]byte(os.Getenv("DAWG_TEST_SUITE_SECRET")))
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
