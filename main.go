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
	"errors"
	"flag"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const indexHTML = `<!DOCTYPE html>
<!-- Polyfill for the old Edge browser -->
<script src="https://cdn.jsdelivr.net/npm/text-encoding@0.7.0/lib/encoding.min.js"></script>
<script src="wasm_exec.js"></script>
<script>
(async () => {
  const resp = await fetch('main.wasm');
  if (!resp.ok) {
    const pre = document.createElement('pre');
    pre.innerText = await resp.text();
    document.body.appendChild(pre);
  } else {
    const src = await resp.arrayBuffer();
    const go = new Go();
    const result = await WebAssembly.instantiate(src, go.importObject);
    go.argv = {{.Argv}};
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

	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	tmpOutputDir = tmp
	return tmpOutputDir, nil
}

func hasGo111Module(env []string) bool {
	for _, e := range env {
		if strings.HasPrefix(e, "GO111MODULE=") {
			return true
		}
	}
	return false
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
	workdir := "."

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
			fargs := flag.Args()
			argv := make([]string, 0, len(fargs))
			for _, a := range fargs {
				argv = append(argv, `"`+template.JSEscapeString(a)+`"`)
			}
			h := strings.ReplaceAll(indexHTML, "{{.Argv}}", "["+strings.Join(argv, ", ")+"]")
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
			http.ServeFile(w, r, f)
			return
		}
	case "main.wasm":
		if _, err := os.Stat(fpath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if errors.Is(err, fs.ErrNotExist) {
			// go build
			args := []string{"build", "-o", filepath.Join(output, "main.wasm")}
			if *flagTags != "" {
				args = append(args, "-tags", *flagTags)
			}
			if *flagOverlay != "" {
				args = append(args, "-overlay", *flagOverlay)
			}
			if len(flag.Args()) > 0 {
				args = append(args, flag.Args()[0])
			} else {
				args = append(args, ".")
			}
			log.Print("go ", strings.Join(args, " "))
			cmdBuild := exec.Command("go", args...)
			cmdBuild.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
			// If GO111MODULE is not specified explicitly, enable Go modules.
			// Enabling this is for backward compatibility of wasmserve.
			if !hasGo111Module(cmdBuild.Env) {
				cmdBuild.Env = append(cmdBuild.Env, "GO111MODULE=on")
			}
			cmdBuild.Dir = workdir
			out, err := cmdBuild.CombinedOutput()
			if err != nil {
				log.Print(err)
				log.Print(string(out))
				http.Error(w, string(out), http.StatusInternalServerError)
				return
			}
			if len(out) > 0 {
				log.Print(string(out))
			}

			f, err := os.Open(filepath.Join(output, "main.wasm"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer f.Close()
			http.ServeContent(w, r, "main.wasm", time.Now(), f)
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

func main() {
	flag.Parse()
	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(*flagHTTP, nil))
}
