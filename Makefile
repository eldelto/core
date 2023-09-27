.DELETE_ON_ERROR:

# High-level commands
.PHONY: build lint
build:
	go build -ldflags="-s -w" -o bin/ github.com/eldelto/core/...

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
