GO_TEST_FLAGS ?= -count=1 -v
BIN_DIR := bin
PLUGIN_BIN := $(BIN_DIR)/kubectl-waitx
PLUGIN_COMPLETE_BIN := $(BIN_DIR)/kubectl_complete-waitx
PREFIX ?= $(HOME)/.local/bin

.PHONY: deps build install test e2e fmt lint clean

deps:
	go mod download
	go mod tidy

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: | $(BIN_DIR)
	rm -f $(PLUGIN_BIN) $(PLUGIN_COMPLETE_BIN)
	go build -o $(PLUGIN_COMPLETE_BIN) .
	install -m 0755 kubectl-waitx $(PLUGIN_BIN)

install: build
	mkdir -p $(PREFIX)
	install -m 0755 $(PLUGIN_COMPLETE_BIN) $(PREFIX)/kubectl_complete-waitx
	install -m 0755 $(PLUGIN_BIN) $(PREFIX)/kubectl-waitx

test: build
	go test ./... $(GO_TEST_FLAGS)

e2e: build
	./hack/e2e/run.sh

fmt:
	go fix ./...
	golangci-lint run --fix ./... || true
	@files="$$(find . -type f -name '*.go' -not -path './.git/*' -not -path './bin/*')"; [ -z "$$files" ] || gofmt -w $$files

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BIN_DIR)
