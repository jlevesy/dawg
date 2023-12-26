package main

import (
	"fmt"
)

//export generate
func generate() uint64 {
	for {
		fmt.Println("poll poll poll, am bad.")
	}
	return 0
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
