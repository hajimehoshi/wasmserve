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
# Be careful that `-tags=example` is required to run the below example application.
wasmserve -tags=example
```

And open `http://localhost:8080/github.com/hajimehoshi/wasmserve/example/` on your browser.

## Example 2

WasmServe can run a local project.

```sh
git clone https://github.com/hajimehoshi/ebiten # This might take several minutes.
cd ebiten/examples/sprites
wasmserve -tags=example
```

And open `http://localhost:8080/` on your browser.

## Known issue with Windows Subsystem for Linux (WSL)

This application sometimes does not work under WSL, due to bugs in WSL, see https://github.com/hajimehoshi/wasmserve/issues/5 for details.
