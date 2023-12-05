.DELETE_ON_ERROR:

# High-level commands
.PHONY: all
all: lint test build

.PHONY: build
build:
	go install -ldflags="-s -w" github.com/eldelto/core/cmd/...

.PHONY: init
init: download
	cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %

.PHONY: download
download:
	go mod download

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	go mod tidy
	go fmt ./...
	staticcheck ./...

.PHONY: test-loop
test-loop:
	echo Waiting for file changes ...
	reflex -r "\.(go|html|css|js)$$" -R ".*node_modules.*" make test

.PHONY: clean
clean:
	rm -f bin/*

.FORCE:
