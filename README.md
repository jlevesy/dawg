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

### What this is right now?

Currently a PoC that reads and executes a statically compiled WASM binary from filesystem, pushes it to grafana.

Building the generators (you'll need tinygo)

```bash
make generators
```

Running the provisioner

```bash
go run ./cmd/provision -generator .dist/simple.wasm -config generator/simple/config.yaml -grafana-url=http://yourgrafanainstance  -grafana-token "yourtoken"
```

### Inspiration

Based on projects built by @K-Phoen:

- [dark](https://github.com/k-phoen/dark)
- [foundation-sdk](https://github.com/grafana/grafana-foundation-sdk
