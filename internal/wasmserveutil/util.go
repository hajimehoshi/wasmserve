// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Hajime Hoshi

package wasmserveutil

import (
	"fmt"
	"regexp"
	"strconv"
)

var reGoVersion = regexp.MustCompile(`^go(\d+)\.(\d+)(\.(\d+))?`)

// WasmExecJSURL returns the URL of wasm_exec.js for the given Go version like 'go1.23.4'.
func WasmExecJSURL(goVersion string) (string, error) {
	m := reGoVersion.FindStringSubmatch(goVersion)
	if len(m) == 0 {
		return "", fmt.Errorf("wasmserve: invalid Go version: %s", goVersion)
	}
	minor, _ := strconv.Atoi(m[2])

	// go.mod might have a version without `.0` like `go1.22`. This version might not work as a part of URL.
	if minor >= 22 && m[3] == "" {
		goVersion += ".0"
	}

	// The directory name was changed from `misc` to `lib` at Go 1.23.
	dir := "lib"
	if minor <= 23 {
		dir = "misc"
	}

	return fmt.Sprintf("https://go.googlesource.com/go/+/refs/tags/%s/%s/wasm/wasm_exec.js?format=TEXT", goVersion, dir), nil
}
