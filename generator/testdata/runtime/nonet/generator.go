package main

import (
	"io"
	"os"

	"github.com/jlevesy/dawg/gdk"
	"github.com/stealthrocket/net/wasip1"
)

//export generate
func generate() uint64 {
	url, err := os.ReadFile(gdk.InputPath)
	if err != nil {
		return gdk.Error(err)
	}

	c, err := wasip1.Dial("tcp", string(url))
	if err != nil {
		return gdk.Error(err)
	}

	b, err := io.ReadAll(c)
	if err != nil {
		return gdk.Error(err)
	}

	return gdk.WriteOutput(b)
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
