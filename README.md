# WasmServe

An HTTP server for Wasm testing like `gopherjs serve`

## Installation

```sh
go install github.com/hajimehoshi/wasmserve@latest
```

## Usage

```
Usage of wasmserve
  -allow-origin string
        Allow specified origin (or * for all origins) to make requests to this server
  -http string
        HTTP bind address to serve (default ":8080")
  -overlay string
        Overwrite source files with a JSON file (see https://pkg.go.dev/cmd/go for more details)
  -tags string
        Build tags
```

## Trigger Refresh

Once the browser loads the page, you can trigger a reload by making a call to teh server at `/_notify`, like this:

```sh
curl localhost:8080/_notify
```

This will make the browser reload. You can add this command to a build script or to an IDE command, to have the browser automatically update without leaving your IDE.

## Example

Running a remote package

```sh
wasmserve github.com/hajimehoshi/wasmserve/example
```

And open `http://localhost:8080/` on your browser.

## Example 2

Running a local package

```sh
git clone https://github.com/hajimehoshi/ebiten # This might take several minutes.
cd ebiten
wasmserve ./examples/sprites
```

And open `http://localhost:8080/` on your browser.

## Known issue with Windows Subsystem for Linux (WSL)

This application sometimes does not work under WSL, due to bugs in WSL, see https://github.com/hajimehoshi/wasmserve/issues/5 for details.

## Tips

* If you want to change the working directory to serve, you can use cd with parentheses:

```
(cd /path/to/working/dir; wasmserve github.com/yourname/yourpackage)
```

* To trigger a browser reload from a script, make a call to `/_notify`, like this:

```sh
curl http://localhost:8080/_notify
```
This will make the browser refresh the page.
