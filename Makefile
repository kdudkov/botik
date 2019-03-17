.PHONY: clean build prepare all test

default: all

all: prepare test build

GIT_REVISION=$(shell git rev-parse --short HEAD)
GIT_BRANCH=$(shell git rev-parse --symbolic-full-name --abbrev-ref HEAD)

LDFLAGS=-ldflags "-s -X main.gitRevision=$(GIT_REVISION) -X main.gitBranch=$(GIT_BRANCH)"

clean:
	rm bin/*

prepare:
	go mod tidy

test:
	go test -v ./...

run:
	go run .

build:
	go build $(LDFLAGS) -o bin/botik
