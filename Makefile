##@ Development

.PHONY: generate
generate: generate_manifests generate_code generate_wasm

.PHONY: generate_manifests
generate_manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=dawg-controller-role crd paths="./..." output:crd:artifacts:config=k8s/crd output:rbac:artifacts:config=k8s/dawg

.PHONY: generate_code
generate_code: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: generate_wasm
generate_wasm:
	go generate -v ./...

.PHONY: clean_generate_wasm
clean_generate_wasm:
	cd ./generator/testdata/runtime && rm -f *.wasm

.PHONY: test
test: generate_wasm
	go test -count=1 -v -timeout=5m -race -cover -run=$(T) ./...

##@ Generators

TINYGO=tinygo
GENERATOR_DIST=$(PWD)/dist/generators

generators: clean_generators $(GENERATOR_DIST) build_generators

$(GENERATOR_DIST):
	mkdir -p $(GENERATOR_DIST)

.PHONY: build_generators
build_generators: example/*
	for dir in $^; do \
		cd $${dir} && $(TINYGO) build -o $(GENERATOR_DIST)/$$(basename "$${dir}").wasm -scheduler=none --no-debug -target wasi ./generator.go ; \
	done

.PHONY: clean_generators
clean_generators:
	rm -rf $(GENERATOR_DIST)

##@ Dev environment

K3S_VERSION?=v1.27.3-k3s1
DEV_CLUSTER_PORT?=3000
DEV_CLUSTER_NAME?=dawg-dev
VERSION?=unknown

.PHONY: dev
dev: preflight_dev generate create_cluster install deploy_dependencies deploy

.PHONY: create_cluster
create_cluster: ## run a local k3d cluster
	k3d cluster create \
		--image="rancher/k3s:$(K3S_VERSION)" \
		-p "$(DEV_CLUSTER_PORT):80@loadbalancer" \
		--registry-create=$(DEV_CLUSTER_NAME).localhost:0.0.0.0:5000 \
		$(DEV_CLUSTER_NAME)

.PHONY: delete_cluster
delete_cluster: ## tears down the k3d dev cluster
	k3d cluster delete $(DEV_CLUSTER_NAME)

.PHONY: install
install: generate_manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f k8s/crd

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config
	kubectl delete -f k8s/crd

.PHONY: deploy
deploy: generate_code generate_manifests
	kubectl kustomize k8s/dawg | VERSION=$(VERSION) KO_DOCKER_REPO=dawg-dev.localhost:5000 ko apply -f -

.PHONY: undeploy
undeploy: generate_manifests
	kubectl kustomize k8s/dawg | kubectl delete -f -

.PHONY: deploy_dependencies
deploy_dependencies: deploy_grafana deploy_prometheus deploy_ksb ## deploy all the dependencies

.PHONY: deploy_grafana
deploy_grafana:
	kubectl apply -k k8s/grafana

.PHONY: deploy_prometheus
deploy_prometheus:
	kubectl apply -k k8s/prometheus

.PHONY: deploy_ksb
deploy_ksb:
	kubectl apply -k k8s/kube-state-metrics

.PHONY: preflight_dev
preflight_dev: ## Checks that all the necesary binaries are present
	@k3d version >/dev/null 2>&1 || (echo "ERROR: k3d is required."; exit 1)
	@kubectl version --client >/dev/null 2>&1 || (echo "ERROR: kubectl is required."; exit 1)
	@ko version >/dev/null 2>&1 || (echo "ERROR: ko is required."; exit 1)
	@grep -Fq "$(DEV_CLUSTER_NAME).localhost" /etc/hosts || (echo "ERROR: please add the following line '127.0.0.1 $(DEV_CLUSTER_NAME).localhost' to your /etc/hosts file"; exit 1)

## Tool Binaries
## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

KUBECTL ?= kubectl
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

CONTROLLER_TOOLS_VERSION ?= v0.13.0

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
