
default: all

.PHONY: all
all: prepare test build

GIT_REVISION=$(shell git rev-parse --short HEAD)
GIT_BRANCH=$(shell git rev-parse --symbolic-full-name --abbrev-ref HEAD)

LDFLAGS=-ldflags "-s -X main.gitRevision=$(GIT_REVISION) -X main.gitBranch=$(GIT_BRANCH)"

.PHONY: clean
clean:
	rm bin/*

.PHONY: prepare
prepare:
	go mod tidy

.PHONY: test
test: prepare
	go test -v ./...

.PHONY: run
run: prepare
	go run .

.PHONY: build
build: prepare
	go build $(LDFLAGS) -o bin/botik
