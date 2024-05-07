.PHONY: build
build:
		go build -v ./cmd/apiserver

.PHONY: test
test:
		go test -v ./internal/app/store -args -n schus -p 19schus78

.DEFAULT_GOAL := build
