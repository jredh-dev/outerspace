# outerspace — human-only tooling monorepo
#
# This repo contains CLIs and scripts that are NOT part of the agent loop.
# OpenCode should never work in this repo directly.

GOBIN ?= $(HOME)/go/bin

.PHONY: build install test clean

build:
	go build ./...

install: build
	go install ./cmd/watcher-ctl/

test:
	go test ./...

clean:
	go clean ./...
