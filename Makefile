BINARY    := ds4-tui
VERSION   ?= 1.0.5
PREFIX    ?= $(HOME)/.local
BIN_DIR   := bin
GOFLAGS   ?=
LDFLAGS   := -s -w -X main.version=$(VERSION)

.PHONY: all build deps verify-deps test race vet run demo doctor setup-led install uninstall clean

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) ./cmd/ds4-tui

deps:
	go mod download

verify-deps:
	go mod verify

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

run:
	go run ./cmd/ds4-tui

demo:
	go run ./cmd/ds4-tui --demo

doctor:
	@command -v go >/dev/null || { echo "Missing: Go 1.24+"; exit 1; }
	go run ./cmd/ds4-tui --doctor

setup-led:
	sudo ./scripts/install-udev.sh

install: build
	install -Dm755 $(BIN_DIR)/$(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY)

clean:
	rm -rf $(BIN_DIR)
