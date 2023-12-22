package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/jlevesy/dawg/generator"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		generatorURL string
	)

	flag.StringVar(&generatorURL, "generator", "", "Path to the WASM binary of the generator")
	flag.Parse()

	if generatorURL == "" || len(flag.Args()) == 0 {
		fmt.Println("Must provide a generatorURL and a binary path")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	store, err := generator.DefaultStore()
	if err != nil {
		fmt.Println("could not build default generator stores", err)
		return 1
	}

	parsedGeneratorURL, err := url.Parse(generatorURL)
	if err != nil {
		fmt.Println("could not parse generator url", err)
		return 1
	}

	genBytes, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Println("could not read generator file", err)
		return 1
	}

	if err := store.Store(ctx, parsedGeneratorURL, &generator.Generator{Bin: genBytes}); err != nil {
		fmt.Println("could not push generator", err)
		return 1
	}

	fmt.Println("Successfully pushed generator", parsedGeneratorURL.String())

	return 0
}
