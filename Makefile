.PHONY: build
build:
		go build -v ./cmd/apiserver

.PHONY: test
test:
		go test -v ./internal/app/store -args -n name -p password -dbip DBServerIP
		go test -v ./internal/app/cmdexec -args -n name -p password -dbip DBServerIP

.DEFAULT_GOAL := build
