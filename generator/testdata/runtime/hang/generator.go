package main

import "time"

//export generate
func generate() uint64 {
	time.Sleep(1000 * time.Second)
	return 0
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
