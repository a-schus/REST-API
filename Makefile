.PHONY: build
build:
		go build -v ./cmd/apiserver

.PHONY: test
test:
		go test -v ./...

.DEFAULT_GOAL := build
