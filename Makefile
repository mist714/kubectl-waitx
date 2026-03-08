GO_TEST_FLAGS ?= -count=1 -v
BIN_DIR := bin
PLUGIN_BIN := $(BIN_DIR)/kubectl-waitx
PLUGIN_COMPLETE_BIN := $(BIN_DIR)/kubectl_complete-waitx

.PHONY: deps build test fmt lint clean

deps:
	go mod download
	go mod tidy

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: | $(BIN_DIR)
	go build -o $(PLUGIN_BIN) .
	printf '%s\n' '#!/bin/sh' 'exec kubectl-waitx --complete -- "$$@"' > $(PLUGIN_COMPLETE_BIN)
	chmod +x $(PLUGIN_COMPLETE_BIN)

test: build
	go test ./... $(GO_TEST_FLAGS)

fmt:
	go fix ./...
	golangci-lint run --fix ./... || true
	@files="$$(find . -type f -name '*.go' -not -path './.git/*' -not -path './bin/*')"; [ -z "$$files" ] || gofmt -w $$files

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BIN_DIR)
