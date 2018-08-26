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
go run github.com/hajimehoshi/wasmserve -tags=example
```

And open `http://localhost:8080/github.com/hajimehoshi/wasmserve/example/` on your browser.

## Example 2

```sh
# Install some libraries.
go get github.com/hajimehoshi/wasmserve
go get github.com/hajimehoshi/ebiten
go get github.com/gopherjs/gopherwasm

# Run WasmServe with Go 1.11 or later.
go run github.com/hajimehoshi/wasmserve -tags=example
```

And open `http://localhost:8080/github.com/hajimehoshi/ebiten/examples/sprites/` on your browser.
