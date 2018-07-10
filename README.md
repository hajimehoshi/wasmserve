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
# Build the latest Go.
git clone https://go.googlesource.com/go go-code
cd go-code/src
./make.sh

# Run WasmServe with the latest Go.
~/go-code/bin/go run github.com/hajimehoshi/wasmserve -tags=example
```

And open `http://localhost:8080/github.com/hajimehoshi/wasmserve/example/` on your browser.

## Example 2

```sh
# Install the latest Go like above.

# Install some libraries.
go get github.com/hajimehoshi/wasmserve
go get github.com/hajimehoshi/ebiten
go get github.com/gopherjs/gopherwasm

# Run WasmServe with the latest Go.
~/go-code/bin/go run github.com/hajimehoshi/wasmserve -tags=example
```

And open `http://localhost:8080/github.com/hajimehoshi/ebiten/examples/sprites/` on your browser.
