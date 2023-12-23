##@ Dev

.PHONY: generate
generate:
	go generate -v ./...

.PHONY: test
test: generate
	go test -count=1 -timeout=5m -race -cover ./...

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

.PHONY: dev
dev: preflight_dev create_cluster deploy_dependencies

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


