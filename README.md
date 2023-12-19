### DAWG: Dashboards as Webassembly Generators

DAWG aims to be Kubernetes controller that allows to provision grafana resouces based on webassembly payload generators.

The motivation here is that the current dashboard model is complex to work with and maintain. Instead of exposing this complexity to the end-user, we would like to create dashboard abstractions using custom codes and only expose the configuration of those dashboards. Doing this, we're hiding away the complexity of the dashboard model to only focus on the essential configuration for a specific dashboard.

For example, we could create a `DeploymentDashboard` webassembly generator that generates a dashboard to view the current state of a deployment. The only configuration necessary would be the namespace and the name of the monitored deployment.

Webassembly provides us with the following advantages:

- Language agnostic(ish, as long as call conventions are implemented in respected) runtime, you could write your generators using any langages that compiles to WASM and supports WASI.
- Allows us to distribute WASM modules in an [OCI Registry](https://github.com/engineerd/wasm-to-oci)
- Dashboards generators can be reused and have their own lifecycle.

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

### What this is right now?

Currently a PoC that reads and executes a statically compiled WASM binary loaded from the filesystem or an OCI registry, pushes it to grafana.

Building the generators (you'll need tinygo)

```bash
make generators
```

Running the provisioner

```bash
# From a local wasm file
go run ./cmd/provision -generator "file://${PWD}/.dist/simple.wasm" -config ./example/simple/config.yaml -grafana-url=http://yourgrafanainstance  -grafana-token "yourtoken"

# From a registry
go run ./cmd/provision -generator "oci://youregistry.domain/reponame/geneatorname:tag" -config ./example/simple/config.yaml -grafana-url=http://yourgrafanainstance  -grafana-token "yourtoken"
```

Pushing a wasm binary to an OCI registry is done using [wasm-to-oci](https://github.com/engineerd/wasm-to-oci), but I'll bring that in house asap.

```bash
wasm-to-oci push .dist/simple.wasm  some-registry:5000/generators/simple:v0.0.1 --use-http
``

### Inspiration

Based on projects built by @K-Phoen:

- [dark](https://github.com/k-phoen/dark)
- [foundation-sdk](https://github.com/grafana/grafana-foundation-sdk


WASM to OCI:

- [wasm-to-oci](https://github.com/engineerd/wasm-to-oci)
