## DAWG: Dashboards as Webassembly Generators

### What this aims to be

#### Motivation

DAWG aims to be Kubernetes controller that allows to provision grafana resouces based on webassembly payload generators.

The motivation here is that the current dashboard model is complex to work with and maintain. Instead of exposing this complexity to the end-user, we would like to create dashboard abstractions using custom code and only expose the configuration of those abstractions to the end user. We're hiding away the complexity of the dashboard model to only focus on the essential configuration for a specific dashboard, all the nitty gritty details are handled by the generators.

For example, we could create a `DeploymentDashboard` generator that builds a dashboard to view the current state of a deployment in Kubernetes. The only configuration necessary would be the namespace and the name of the monitored deployment.

The manifest would look like the following:

```yaml
apiVersion: dawg.urcloud.cc
kind: Dashboard
meta:
  name: super-dashboard
  namespace: app
spec:
  generator: oci://registry.domain/deployment-dashboard:v3.45.0
  config: | # arbitrary config passed to the wasm binary.
    namespace: foo
    name: super-deployment
```

You can find a generator example [here](./example/simple)

#### Why WASM?

Webassembly provides us with the following advantages:

- Language agnostic(ish, as long as call conventions are implemented) runtime, you could write your generators using any langages that compiles to WASM and supports WASI.
- Could be distributed using an [OCI Registry](https://github.com/engineerd/wasm-to-oci)
- Dashboards generators can be reused and have their own lifecycle.

### What this is right now

Currently a PoC that reads and executes a compiled WASM binary loaded from the filesystem or an OCI registry and pushes it to grafana.

I'm currently exploring the problem space as I'm both unfamilliar with WASM and OCI in depth.
This is all very early stage, much TODO, very YOLO.

Building the examples generators (you'll need tinygo). This will write the built generrators into `./dist/generators` by default.

```bash
make generators
```

Applyging a Generated Dashboard to Grafana

```bash
# From a local wasm file
go run ./cmd/apply -generator "file://${PWD}/dist/generators/simple.wasm" -config ./example/simple/config.yaml -grafana-url=http://yourgrafanainstance  -grafana-token "yourtoken"

# From a registry
go run ./cmd/provision -generator "registry://youregistry.domain/reponame/generatorname:tag" -config ./example/simple/config.yaml -grafana-url=http://yourgrafanainstance  -grafana-token "yourtoken"
```

Pushing a generator to a registry:

```bash
go run ./cmd/push -generator registry://registry.domain/remponame/generratorname:tag dist/generators/simple.wasm
```

### Development Environment

It comes with a basic developlent environment that creates a k8s cluster and provisions Grafana, Prometheus and a few exporters. It also provisions a registry on port `:5000`.

You can run it using `make dev`.

### Resources

Based on projects built by [K-Phoen](https://github.com/k-phoen/):

- [dark](https://github.com/k-phoen/dark)
- [foundation-sdk](https://github.com/grafana/grafana-foundation-sdk)

WASM:

- [wazero articles from k33g](https://k33g.hashnode.dev/series/wazero-first-steps)
- [wasm-to-oci](https://github.com/engineerd/wasm-to-oci)
