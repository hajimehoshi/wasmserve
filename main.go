// Copyright 2018 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const mainWasm = "main.wasm"

const indexHTML = `<!DOCTYPE html>
<script src="wasm_exec.js"></script>
<script>
(async () => {
  const resp = await fetch({{.MainWasm}});
  if (!resp.ok) {
    const pre = document.createElement('pre');
    pre.innerText = await resp.text();
    document.body.appendChild(pre);
  } else {
    const src = await resp.arrayBuffer();
    const go = new Go();
    const result = await WebAssembly.instantiate(src, go.importObject);
    go.argv = {{.Argv}};
    go.env = {{.Env}};
    go.run(result.instance);
  }
  const reload = await fetch('_wait');
  // The server sends a response for '_wait' when a request is sent to '_notify'.
  if (reload.ok) {
    location.reload();
  }
})();
</script>
`

var (
	flagHTTP        = flag.String("http", ":8080", "HTTP bind address to serve")
	flagTags        = flag.String("tags", "", "Build tags")
	flagAllowOrigin = flag.String("allow-origin", "", "Allow specified origin (or * for all origins) to make requests to this server")
	flagOverlay     = flag.String("overlay", "", "Overwrite source files with a JSON file (see https://pkg.go.dev/cmd/go for more details)")
)

var (
	tmpOutputDir = ""
	waitChannel  = make(chan struct{})
)

func ensureTmpOutputDir() (string, error) {
	if tmpOutputDir != "" {
		return tmpOutputDir, nil
	}

	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}
	tmpOutputDir = tmp
	return tmpOutputDir, nil
}

func handle(w http.ResponseWriter, r *http.Request) {
	if *flagAllowOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", *flagAllowOrigin)
	}

	output, err := ensureTmpOutputDir()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	upath := r.URL.Path[1:]
	fpath := path.Base(upath)

	if !strings.HasSuffix(r.URL.Path, "/") {
		fi, err := os.Stat(fpath)
		if err != nil && !os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if fi != nil && fi.IsDir() {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusSeeOther)
			return
		}
	}

	switch filepath.Base(fpath) {
	case ".":
		fpath = filepath.Join(fpath, "index.html")
		fallthrough
	case "index.html":
		if _, err := os.Stat(fpath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if errors.Is(err, fs.ErrNotExist) {
			firstArg := filepath.Join(output, mainWasm)
			fargs := make([]string, flag.NArg())
			copy(fargs, flag.Args())
			if len(fargs) == 0 {
				fargs = append(fargs, firstArg)
			} else {
				fargs[0] = firstArg
			}
			argv := make([]string, 0, len(fargs))
			for _, a := range fargs {
				argv = append(argv, `"`+template.JSEscapeString(a)+`"`)
			}
			h := strings.ReplaceAll(indexHTML, "{{.Argv}}", "["+strings.Join(argv, ", ")+"]")

			oenv := os.Environ()
			env := make([]string, 0, len(oenv))
			for _, e := range oenv {
				split := strings.SplitN(e, "=", 2)
				env = append(env, `"`+template.JSEscapeString(split[0])+`": "`+template.JSEscapeString(split[1])+`"`)
			}
			h = strings.ReplaceAll(h, "{{.Env}}", "{"+strings.Join(env, ", ")+"}")

			h = strings.ReplaceAll(h, "{{.MainWasm}}", `"`+template.JSEscapeString(mainWasm)+`"`)

			http.ServeContent(w, r, "index.html", time.Now(), bytes.NewReader([]byte(h)))
			return
		}
	case "wasm_exec.js":
		if _, err := os.Stat(fpath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if errors.Is(err, fs.ErrNotExist) {
			out, err := exec.Command("go", "env", "GOROOT").Output()
			if err != nil {
				log.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			f := filepath.Join(strings.TrimSpace(string(out)), "misc", "wasm", "wasm_exec.js")
			if _, err := os.Stat(f); err == nil {
				http.ServeFile(w, r, f)
				return
			}

			// wasm_exec.js might not be found when the current Go toolchain is automatically downloaded [1].
			// In this case, wasm_exec.js should be obtained externally.
			//
			// [1] https://go.dev/doc/toolchain
			content, err := fetchWasmExecJS()
			if err != nil {
				log.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.ServeContent(w, r, "wasm_exec.js", time.Time{}, bytes.NewReader(content))
			return
		}
	case mainWasm:
		if _, err := os.Stat(fpath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if errors.Is(err, fs.ErrNotExist) {
			if err := goBuild(filepath.Join(output, mainWasm)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			f, err := os.Open(filepath.Join(output, mainWasm))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer f.Close()

			http.ServeContent(w, r, mainWasm, time.Now(), f)
			return
		}
	case "_wait":
		waitForUpdate(w, r)
		return
	case "_notify":
		notifyWaiters(w, r)
		return
	}

	http.ServeFile(w, r, filepath.Join(".", r.URL.Path))
}

func target() string {
	if flag.NArg() > 0 {
		return flag.Args()[0]
	}
	return "."
}

// goVersion fetches the current using Go's version.
// goVersion is different from runtime.Version(), which returns a Go version for this wasmserve build.
func goVersion() (string, error) {
	cmd := exec.Command("go", "list", "-f", "go{{.Module.GoVersion}}", target())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if stderr.Len() > 0 {
		log.Print(stderr.String())
	}

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s%w", stderr.String(), err)
	}

	return strings.TrimSpace(string(out)), nil
}

func goBuild(outputPath string) error {
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return err
	}

	// buildDir is the directory to build the Wasm binary.
	buildDir := "."

	// If the target path is absolute, an environment with go.mod is required.
	t := target()
	if !strings.HasPrefix(t, "./") && !strings.HasPrefix(t, ".\\") && t != "." {
		dir, err := os.MkdirTemp("", "")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
		buildDir = dir

		// Run `go mod init`.
		cmd := exec.Command("go", "mod", "init", "foo")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if len(out) > 0 {
			log.Print(string(out))
		}
		if err != nil {
			return err
		}

		// Run `go get`.
		cmd = exec.Command("go", "get", t)
		cmd.Dir = dir
		out, err = cmd.CombinedOutput()
		if len(out) > 0 {
			log.Print(string(out))
		}
		if err != nil {
			return err
		}

		// `go build` cannot accept a path with a version. Drop it.
		if idx := strings.LastIndex(t, "@"); idx >= 0 {
			t = t[:idx]
		}
	}

	// Run `go build`.
	args := []string{"build"}
	if *flagTags != "" {
		args = append(args, "-tags", *flagTags)
	}
	if *flagOverlay != "" {
		args = append(args, "-overlay", *flagOverlay)
	}
	args = append(args, "-o", absOutputPath)
	args = append(args, t)
	log.Print("go ", strings.Join(args, " "))

	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Dir = buildDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if stdout.Len() > 0 {
		log.Print(stdout.String())
	}
	if stderr.Len() > 0 {
		log.Print(stderr.String())
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s%w", stderr.String(), err)
	}
	return nil
}

func fetchWasmExecJS() ([]byte, error) {
	v, err := goVersion()
	if err != nil {
		return nil, err
	}

	// TODO: Cache the result.

	url := fmt.Sprintf("https://go.googlesource.com/go/+/refs/tags/%s/misc/wasm/wasm_exec.js?format=TEXT", v)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, resp.Body))
	if err != nil {
		return nil, err
	}

	return content, nil
}

func waitForUpdate(w http.ResponseWriter, r *http.Request) {
	waitChannel <- struct{}{}
	http.ServeContent(w, r, "", time.Now(), bytes.NewReader(nil))
}

func notifyWaiters(w http.ResponseWriter, r *http.Request) {
	for {
		select {
		case <-waitChannel:
		default:
			http.ServeContent(w, r, "", time.Now(), bytes.NewReader(nil))
			return
		}
	}
}

func Run() {
	flag.Parse()

	tmpDir, err := ensureTmpOutputDir()
	if err != nil {
		log.Fatalf("Failed to create temporary directory: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	var server http.Server

	shutdown := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		log.Printf("Shutting down server...")

		// Received an interrupt signal, shut down.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		err := server.Shutdown(ctx)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Error at server.Shutdown: %v", err)
		}
		close(shutdown)

		<-sigint
		// Hard exit on the second ctrl-c.
		os.Exit(0)
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handle)
	server.Handler = mux
	server.Addr = *flagHTTP

	log.Printf("Listening on %v", *flagHTTP)
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Error at server.ListenAndServe: %v", err)
	}

	<-shutdown

	log.Printf("Exiting")
}
