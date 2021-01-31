# WasmServe

An HTTP server for Wasm testing like `gopherjs serve`

## Installation

```sh
go get -u github.com/hajimehoshi/wasmserve
```

## Usage

```
Usage of wasmserve
  -allow-origin string
        Allow specified origin (or * for all origins) to make requests to this server
  -http string
        HTTP bind address to serve (default ":8080")
  -tags string
        Build tags
  -workdir specify the build workdir path
```

## Example

Running a remote package

```sh
# Be careful that `-tags=example` is required to run the below example application.
wasmserve -tags=example github.com/hajimehoshi/wasmserve/example
```

And open `http://localhost:8080/` on your browser.

## Example 2

Running a local package

```sh
git clone https://github.com/hajimehoshi/ebiten # This might take several minutes.
cd ebiten
wasmserve -tags=example ./examples/sprites
```

And open `http://localhost:8080/` on your browser.

## Known issue with Windows Subsystem for Linux (WSL)

This application sometimes does not work under WSL, due to bugs in WSL, see https://github.com/hajimehoshi/wasmserve/issues/5 for details.
