//go:build tools

package main

import (
	_ "github.com/cespare/reflex"
	_ "honnef.co/go/tools/cmd/staticcheck" // TODO: Replace with golangci-lint
)
