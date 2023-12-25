package main

import (
	"os"

	"github.com/jlevesy/dawg/gdk"
)

//export generate
func generate() uint64 {
	path, err := os.ReadFile(gdk.InputPath)
	if err != nil {
		return gdk.Error(err)
	}

	b, err := os.ReadFile(string(path))
	if err != nil {
		return gdk.Error(err)
	}

	return gdk.WriteOutput(b)
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
