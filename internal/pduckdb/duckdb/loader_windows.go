//go:build windows

package duckdb

import "golang.org/x/sys/windows"

// openLibrary loads a DLL on Windows via the Win32 LoadLibrary API and returns a
// handle usable by purego.RegisterLibFunc. This is the Windows support added on
// top of go-pduckdb (which only implemented the Unix dlopen path). It keeps the
// driver CGO-free: golang.org/x/sys/windows uses the syscall package directly.
func openLibrary(path string) (uintptr, error) {
	h, err := windows.LoadLibrary(path)
	if err != nil {
		return 0, err
	}
	return uintptr(h), nil
}
