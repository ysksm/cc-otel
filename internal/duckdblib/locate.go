// Package duckdblib locates the native DuckDB shared library at runtime.
//
// The Go side of cc-otel is fully CGO-free and cross-compiles to every target,
// but DuckDB itself is a native C++ library that must be present at run time.
// This package finds an existing libduckdb on the system or downloads the
// matching official release once and caches it, then points the purego loader
// at it via the DUCKDB_LIBRARY_PATH environment variable.
package duckdblib

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Version is the DuckDB release the bundled binding is tested against.
const Version = "v1.4.1"

// libFileName is the shared-library file name inside the release archive, per OS.
func libFileName() string {
	switch runtime.GOOS {
	case "windows":
		return "duckdb.dll"
	case "darwin":
		return "libduckdb.dylib"
	default:
		return "libduckdb.so"
	}
}

// assetName returns the DuckDB release asset (zip) for the current platform.
func assetName() (string, error) {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "libduckdb-linux-amd64.zip", nil
		case "arm64":
			return "libduckdb-linux-arm64.zip", nil
		}
	case "darwin":
		// DuckDB ships a universal binary covering amd64 + arm64.
		return "libduckdb-osx-universal.zip", nil
	case "windows":
		if runtime.GOARCH == "amd64" {
			return "libduckdb-windows-amd64.zip", nil
		}
	}
	return "", fmt.Errorf("no prebuilt DuckDB library for %s/%s; set DUCKDB_LIBRARY_PATH to a libduckdb you provide", runtime.GOOS, runtime.GOARCH)
}

// cacheDir is where downloaded libraries are stored, namespaced by version+platform.
func cacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "cc-otel", "duckdb-"+Version+"-"+runtime.GOOS+"-"+runtime.GOARCH), nil
}

// Ensure returns a path to a usable libduckdb, downloading it if necessary, and
// exports DUCKDB_LIBRARY_PATH so the purego loader picks it up. If the caller has
// already set DUCKDB_LIBRARY_PATH, that value is respected and returned as-is.
func Ensure() (string, error) {
	if p := os.Getenv("DUCKDB_LIBRARY_PATH"); p != "" {
		return p, nil
	}

	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	libPath := filepath.Join(dir, libFileName())
	if fi, err := os.Stat(libPath); err == nil && fi.Size() > 0 {
		os.Setenv("DUCKDB_LIBRARY_PATH", libPath)
		return libPath, nil
	}

	if err := download(dir); err != nil {
		return "", err
	}
	if _, err := os.Stat(libPath); err != nil {
		return "", fmt.Errorf("downloaded archive did not contain %s: %w", libFileName(), err)
	}
	os.Setenv("DUCKDB_LIBRARY_PATH", libPath)
	return libPath, nil
}

// download fetches and extracts the platform DuckDB release archive into dir.
func download(dir string) error {
	asset, err := assetName()
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://github.com/duckdb/duckdb/releases/download/%s/%s", Version, asset)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download DuckDB library: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download DuckDB library: unexpected status %s for %s", resp.Status, url)
	}

	tmp, err := os.CreateTemp(dir, "libduckdb-*.zip")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	return extract(tmpName, dir)
}

// extract unzips the library file(s) from the archive into dir (flat).
func extract(zipPath, dir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		name := filepath.Base(f.Name)
		// Only extract shared-library artifacts.
		switch filepath.Ext(name) {
		case ".so", ".dylib", ".dll":
		default:
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return err
		}
		rc.Close()
		out.Close()
		if runtime.GOOS != "windows" {
			os.Chmod(filepath.Join(dir, name), 0o755)
		}
	}
	return nil
}
