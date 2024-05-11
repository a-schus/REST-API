.PHONY: build
build:
		go build -v ./cmd/apiserver

.PHONY: test
test:
		go test -v ./internal/app/store -args -n schus -p 19schus78 -dbip localhost
		go test -v ./internal/app/cmdexec -args -n schus -p 19schus78 -dbip localhost

.DEFAULT_GOAL := build
