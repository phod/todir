VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test vet install

build:
	go build -ldflags "$(LDFLAGS)" -o bin/tkto ./cmd/tkto

test:
	go test ./...

vet:
	go vet ./...

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/tkto
