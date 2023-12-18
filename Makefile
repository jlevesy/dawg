TINYGO=tinygo
GENERATOR_DIST=$(PWD)/.dist

generators: clean_generators $(GENERATOR_DIST) build_generators

$(GENERATOR_DIST):
	mkdir -p $(GENERATOR_DIST)

.PHONY: build_generators
build_generators: generator/*
	for dir in $^; do \
		cd $${dir} && $(TINYGO) build -o $(GENERATOR_DIST)/$$(basename "$${dir}").wasm -scheduler=none --no-debug -target wasi ./generator.go ; \
	done

.PHONY: clean_generators
clean_generators:
	rm -rf $(GENERATOR_DIST)
