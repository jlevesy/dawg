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
	"github.com/jlevesy/dawg/pkg/grafana"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		generatorURL string
		configPath   string
		grafanaURL   string
		grafanaToken string
	)

	flag.StringVar(&generatorURL, "generator", "", "Path to the WASM binary of the generator")
	flag.StringVar(&configPath, "config", "", "Path to the config of the generator")
	flag.StringVar(&grafanaURL, "grafana-url", "", "URL of the grafana instance to provision")
	flag.StringVar(&grafanaToken, "grafana-token", "", "API token to use with the grafana instance")
	flag.Parse()

	if generatorURL == "" || configPath == "" {
		fmt.Println("Must provide a generator path and the config path")
		return 1
	}

	if grafanaURL == "" {
		fmt.Println("Must provide a grafana URL")
		return 1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	store, err := generator.DefaultStore()
	if err != nil {
		fmt.Println("could not build default generator stores", err)
		return 1
	}

	runtime, shutdownRuntime, err := generator.DefaultRuntime(ctx)
	if err != nil {
		fmt.Println("could not setup generator runtime", err)
		return 1
	}

	defer func() {
		err := shutdownRuntime(context.Background())
		if err != nil {
			fmt.Println("could not shutdown runtime", err)
		}
	}()

	parsedGeneratorURL, err := url.Parse(generatorURL)
	if err != nil {
		fmt.Println("could not parse generator url", err)
		return 1
	}

	gen, err := store.Load(ctx, parsedGeneratorURL)
	if err != nil {
		fmt.Println("could not load generator", err)
		return 1
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	dashboardPayload, err := runtime.Execute(ctx, gen, configBytes)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	var grafanaOpts []grafana.ClientOpt

	if grafanaToken != "" {
		grafanaOpts = append(grafanaOpts, grafana.WithAuthToken(grafanaToken))
	}

	grafanaClient := grafana.NewClient(grafanaURL, grafanaOpts...)

	dashboard, err := grafanaClient.CreateDashboard(
		ctx,
		&grafana.CreateDashboardRequest{
			Dashboard: dashboardPayload.Payload,
			Overwrite: true,
		},
	)
	if err != nil {
		fmt.Println("could not create dashboard", err)
		return 1
	}

	fmt.Println("Created dashboard", dashboard)

	return 0
}
