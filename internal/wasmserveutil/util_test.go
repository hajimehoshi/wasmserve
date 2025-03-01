// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Hajime Hoshi

package wasmserveutil_test

import (
	"testing"

	"github.com/hajimehoshi/wasmserve/internal/wasmserveutil"
)

func TestWasmExecJSURL(t *testing.T) {
	testCases := []struct {
		goVersion string
		url       string
		error     bool
	}{
		{
			goVersion: "invalid",
			url:       "",
			error:     true,
		},
		{
			goVersion: "go1.16",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.16/misc/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go1.21",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.21/misc/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go1.22",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.22.0/misc/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go1.22.0",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.22.0/misc/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go1.22.1",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.22.1/misc/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go1.23",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.23.0/misc/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go1.23.0",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.23.0/misc/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go1.23.1",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.23.1/misc/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go1.24",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.24.0/lib/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go1.24.0",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.24.0/lib/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go1.24.1",
			url:       "https://go.googlesource.com/go/+/refs/tags/go1.24.1/lib/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go2.23",
			url:       "https://go.googlesource.com/go/+/refs/tags/go2.23.0/lib/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go2.23.0",
			url:       "https://go.googlesource.com/go/+/refs/tags/go2.23.0/lib/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go2.23.1",
			url:       "https://go.googlesource.com/go/+/refs/tags/go2.23.1/lib/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go2.24",
			url:       "https://go.googlesource.com/go/+/refs/tags/go2.24.0/lib/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go2.24.0",
			url:       "https://go.googlesource.com/go/+/refs/tags/go2.24.0/lib/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
		{
			goVersion: "go2.24.1",
			url:       "https://go.googlesource.com/go/+/refs/tags/go2.24.1/lib/wasm/wasm_exec.js?format=TEXT",
			error:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.goVersion, func(t *testing.T) {
			url, err := wasmserveutil.WasmExecJSURL(tc.goVersion)
			if err != nil && !tc.error {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if err == nil && tc.error {
				t.Errorf("no error")
				return
			}
			if url != tc.url {
				t.Errorf("got: %s, want: %s", url, tc.url)
			}
		})
	}
}
