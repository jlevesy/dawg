module github.com/jlevesy/dawg/generator/simple

go 1.21.5

require (
	github.com/grafana/grafana-foundation-sdk/go v0.0.0-20231215161058-b821de2a2155
	github.com/jlevesy/dawg v0.0.0-20231221151413-e15d579dfea0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/jlevesy/dawg => ./../../
