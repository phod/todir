VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test vet install

build:
	go build -ldflags "$(LDFLAGS)" -o bin/todir ./cmd/todir

test:
	go test ./...

vet:
	go vet ./...

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/todir
