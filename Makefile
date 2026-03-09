BIN_DIR := bin
PLUGIN_BIN := $(BIN_DIR)/kubectl-waitx
PLUGIN_COMPLETE_BIN := $(BIN_DIR)/kubectl_complete-waitx
PREFIX ?= $(HOME)/.local/bin
ASCIINEMA ?= asciinema
SVG_TERM ?= svg-term
GO_SOURCES := main.go $(wildcard internal/cmd/*.go)
ASCIINEMA_SCRIPT := hack/asciinema/readme-demo.sh
ASCIINEMA_CAST := hack/asciinema/readme-demo.cast
ASCIINEMA_SVG := hack/asciinema/readme-demo.svg

.PHONY: deps build install test e2e asciinema-demo fmt lint clean

deps:
	go mod download
	go mod tidy

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: $(PLUGIN_BIN) $(PLUGIN_COMPLETE_BIN)

$(PLUGIN_BIN): $(GO_SOURCES) go.mod go.sum | $(BIN_DIR)
	go build -o $(PLUGIN_BIN) .

$(PLUGIN_COMPLETE_BIN): $(PLUGIN_BIN)
	ln -sf kubectl-waitx $(PLUGIN_COMPLETE_BIN)

install: build
	mkdir -p $(PREFIX)
	install -m 0755 $(PLUGIN_BIN) $(PREFIX)/kubectl-waitx
	ln -sf kubectl-waitx $(PREFIX)/kubectl_complete-waitx

test:
	go test ./... -count=1 -v

e2e: build
	./hack/e2e/run.sh

$(ASCIINEMA_CAST): $(ASCIINEMA_SCRIPT)
	$(ASCIINEMA) record --overwrite --output-format asciicast-v2 --headless --command ./$(ASCIINEMA_SCRIPT) $(ASCIINEMA_CAST)

$(ASCIINEMA_SVG): $(ASCIINEMA_CAST)
	$(SVG_TERM) --in $(ASCIINEMA_CAST) --out $(ASCIINEMA_SVG) --width 100 --height 24 --window --no-cursor

gif: $(ASCIINEMA_SVG)

fmt:
	go fix ./...
	golangci-lint run --fix ./... || true
	@files="$$(find . -type f -name '*.go' -not -path './.git/*' -not -path './bin/*')"; [ -z "$$files" ] || gofmt -w $$files

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BIN_DIR) hack/asciinema/readme-demo.cast
