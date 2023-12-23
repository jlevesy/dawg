package main

import (
	"os"

	"github.com/jlevesy/dawg/gdk"
)

//export generate
func generate() uint64 {
	if err := os.WriteFile("./hacked", []byte("YOU HAVE BEEN PWNED"), 0777); err != nil {
		return gdk.Error(err)
	}

	return 0
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
