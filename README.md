## DAWG: Dashboards as Webassembly Generators

### What this aims to be

DAWG aims to define a new way of delivering reusable and easy to configure Grafana resources (Dashboards, AlertRules...). It boils down to the idea of using a combination of webassembly binaries (embedding the dashboard generation logic) called generators and minimal configuration.

The primary target is Kubernetes, but it also might come as a CLI to allow other use cases.

#### Motivation

The motivation here is that the current Grafana model is complex to work with. Instead of exposing this complexity to the end-user, we would like to create various dashboards abstractions using custom code and only expose the configuration of those abstractions to the end user.

In other words, we're hiding away the complexity of the dashboard model to only focus on the essential configuration for a specific dashboard, all the nitty gritty details are handled by the generators logic.

For instance, we could imagine a dashboard that monitors a Kubernetes deployment. Using DAWG, it would come as a generator that builds the dashboard. This generator would expect a minimal configuration model that could be provided by any users when the dashboard is instanciated.

In the context of Kubernetes, the manifest will look like the following:

```yaml
apiVersion: dawg.urcloud.cc/v1
kind: Dashboard
meta:
  name: super-dashboard
  namespace: app
spec:
  generator: registry://registry.domain/deployment-dashboard:v3.45.0
  config: | # arbitrary config passed to the wasm binary.
    namespace: foo
    name: super-deployment
```

You can find a generator example [here](./example/simple) as well as an actual manifest example [here](k8s/example/example_dashboard.yaml).

#### Delivering Generators

To deliver the generator logic, we chose Webassembly as it provides us with the following advantages:

- Language agnostic(ish, as long as call proper conventions are implemented) runtime, you could write your generators using any langages that compiles to WASM and supports WASI. That being said, we're only supporting `tinygo` at the moment.
- WASM binaries are distribuable using an [OCI Registry](./generator/registry.go), this allows to provide the same way of working that standard container images as well as opening the way to secure the generator delivery using notary for example.
- "Sandboxed & Secure", notice the quotes. I implemended a [collection of (naive) tests](./generator/runtime_test.go) to build up my understanding on that topic a bit, but this should definitely be looked at carefully.

### What this is right now?

Currently a prototype CLI tool as well as a Kubernetes controller that allows provisioning dashboards using a CRD.

#### CLI

The CLI allows that reads and executes a compiled WASM binary loaded from the filesystem or an OCI registry and pushes the generated dashboard manifest to Grafana.

To build the example generators (you'll need tinygo), you need to run the following command. This will write the built generrators into `./dist/generators` by default.

```bash
make generators
```

In order to apply a generated dashboard to Grafana, run the following command:

```bash
# From a local wasm file
go run ./cmd/apply -generator "file://${PWD}/dist/generators/simple" -config ./example/simple/config.yaml -grafana-url=http://yourgrafanainstance  -grafana-token "yourtoken"

# From a registry
go run ./cmd/apply -generator "registry://youregistry.domain/reponame/generatorname:tag" -config ./example/simple/config.yaml -grafana-url=http://yourgrafanainstance  -grafana-token "yourtoken"
```

Pushing a generator to a registry:

```bash
go run ./cmd/push -generator registry://registry.domain/remponame/generratorname:tag dist/generators/simple.wasm
```

#### Kubernetes Controller

The k8s controllers manage a new kind of custom resource called a `Dashboard`. When a new resource is created it reconciliates the expressed state with the managed Grafana instance by fetching the generator, executing it with the given configuration and pushing the generated configuration to Grafana. It also handles deletion.

#### Development environment

It comes with a basic developlent environment that creates a k8s cluster and provisions Grafana, Prometheus and a few exporters. It also provisions a registry on port `:5000`.

You'll need to run the following commands to get environment running:

- `make dev`, this will check that all required binaries are installed as well as your /etc/hosts is configured.
- After a while, head to the [local grafana instance](http://dawg-dev.localhost:3000/grafana) and generate a service account as well as a service account token.
- Then run `GRAFANA_TOKEN=<your token> make set_grafana_token deploy restart` to set up the controller.
- Head back to grafana, you should see an example dashboard created.

### Resources

Based on projects built by [K-Phoen](https://github.com/k-phoen/):

- [dark](https://github.com/k-phoen/dark)
- [foundation-sdk](https://github.com/grafana/grafana-foundation-sdk)

WASM:

- [wazero articles from k33g](https://k33g.hashnode.dev/series/wazero-first-steps)
- [wasm-to-oci](https://github.com/engineerd/wasm-to-oci)

ORAS:

- [oras project](https://oras.land/)
