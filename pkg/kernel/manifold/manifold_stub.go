//go:build !manifold

// Package manifold provides a CGo-based geometry kernel binding to the
// Manifold library. When the "manifold" build tag is not set, this stub
// package is compiled instead, returning an error from New().
//
// Build with: go build -tags=manifold
package manifold

import (
	"errors"

	"github.com/chazu/lignin/pkg/kernel"
)

// New returns an error indicating Manifold is not available.
// Build with -tags=manifold to enable.
func New() (kernel.Kernel, error) {
	return nil, errors.New("manifold kernel not available: build with -tags=manifold")
}
