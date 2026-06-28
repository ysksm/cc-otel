//go:build darwin || linux || freebsd

package duckdb

import "github.com/ebitengine/purego"

// openLibrary loads a shared library on Unix-like systems via dlopen.
func openLibrary(path string) (uintptr, error) {
	return purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}
