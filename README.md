# WasmServe

An HTTP server for Wasm testing like `gopherjs serve`

## Usage

```
Usage of wasmserve
  -http string
        HTTP bind address to serve (default ":8080")
  -tags string
        Build tags
```

## Example

```sh
# Run WasmServe with Go 1.11 or later.
# Be careful that `-tags=example` is required to run the below example application.
wasmserve -tags=example
```

And open `http://localhost:8080/github.com/hajimehoshi/wasmserve/example/` on your browser.

## Example 2

```sh
# Run WasmServe with Go 1.11 or later.
wasmserve -tags=example
```

And open `http://localhost:8080/github.com/hajimehoshi/ebiten/examples/sprites/` on your browser.

Known issue: `wasmserve` tries to get Ebiten ver 1.7.x, which doesn't work with Wasm. I'll fix this to enable to specify the version explicitly at URL.