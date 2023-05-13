.DELETE_ON_ERROR:

GO_CMDS := $(shell find cmd -name "*.go" -not -name "*_test.go")
GO_BINARIES := $(GO_CMDS:cmd/%.go=bin/%)

# High-level commands
.PHONY: build lint
build: $(GO_BINARIES)

.PHONY: init
init: download
	cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %

.PHONY: download
download:
	go mod download

$(GO_BINARIES): bin/%: cmd/%.go
	go build -o bin/ $^

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
