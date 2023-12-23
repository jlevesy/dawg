package main

import (
	"bytes"
	"encoding/json"

	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
	"github.com/jlevesy/dawg/gdk"
	"gopkg.in/yaml.v3"
)

type config struct {
	AppName string `yaml:"app_name"`
}

//export generate
func generate(ptr, size uint32) uint64 {
	var cfg config

	if err := yaml.Unmarshal(gdk.ReadInput(ptr, size), &cfg); err != nil {
		panic(err)
	}

	dashboard, err := dashboard.NewDashboardBuilder(cfg.AppName).
		Uid("generated-from-go").
		Tags([]string{"generated", "from", "go"}).
		Refresh("1m").
		Time("now-30m", "now").
		Timezone(common.TimeZoneBrowser).
		WithRow(dashboard.NewRowBuilder("Overview")).
		WithPanel(
			timeseries.NewPanelBuilder().
				Title("Network Received").
				Unit("bps").
				Min(0).
				WithTarget(
					prometheus.NewDataqueryBuilder().
						Expr(`rate(node_network_receive_bytes_total{job="integrations/raspberrypi-node", device!="lo"}[$__rate_interval]) * 8`).
						LegendFormat("{{ device }}"),
				),
		).
		Build()
	if err != nil {
		panic(err)
	}

	var out bytes.Buffer

	if err := json.NewEncoder(&out).Encode(dashboard); err != nil {
		panic(err)
	}

	return gdk.WriteOutput(out.Bytes())
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
