VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test vet install

build:
	go build -ldflags "$(LDFLAGS)" -o bin/tele ./cmd/tele

test:
	go test ./...

vet:
	go vet ./...

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/tele
