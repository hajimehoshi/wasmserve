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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const indexHTML = `<!DOCTYPE html>
<script src="wasm_exec.js"></script>
<script>
(async () => {
  const resp = await fetch('main.wasm');
  if (!resp.ok) {
    const pre = document.createElement('pre');
    pre.innerText = await resp.text();
    document.body.appendChild(pre);
    return;
  }
  const src = await resp.arrayBuffer();
  const go = new Go();
  WebAssembly.instantiate(src, go.importObject).then(result => {
    go.run(result.instance);
  });
})();
</script>
`

var (
	flagHTTP = flag.String("http", ":8080", "HTTP bind address to serve")
	flagTags = flag.String("tags", "", "Build tags")
)

func handle(w http.ResponseWriter, r *http.Request) {
	src := filepath.Join(runtime.GOROOT(), "src")
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		src = filepath.Join(gopath, "src")
	}
	path := filepath.Join(src, r.URL.Path[1:])

	if !strings.HasSuffix(r.URL.Path, "/") {
		fi, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if fi != nil && fi.IsDir() {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusSeeOther)
			return
		}
	}

	if strings.HasSuffix(r.URL.Path, "/") {
		path = filepath.Join(path, "index.html")
	}

	switch filepath.Base(path) {
	case "index.html":
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeContent(w, r, "index.html", time.Now(), bytes.NewReader([]byte(indexHTML)))
			return
		}
	case "wasm_exec.js":
		if _, err := os.Stat(path); os.IsNotExist(err) {
			f := filepath.Join(runtime.GOROOT(), "misc", "wasm", "wasm_exec.js")
			http.ServeFile(w, r, f)
			return
		}
	case "main.wasm":
		if _, err := os.Stat(path); os.IsNotExist(err) {
			tmp, err := ioutil.TempDir("", "")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out := filepath.Join(tmp, "main.wasm")

			args := []string{"build", "-o", out}
			if *flagTags != "" {
				args = append(args, "-tags", *flagTags)
			}
			args = append(args, filepath.Dir(r.URL.Path[1:]))

			cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), args...)
			cmd.Env = []string{"GOOS=js", "GOARCH=wasm"}
			if gopath := os.Getenv("GOPATH"); gopath != "" {
				cmd.Env = append(cmd.Env, fmt.Sprintf("GOPATH=%s", gopath))
			}
			stderr, err := cmd.CombinedOutput()
			if err != nil {
				log.Print(err)
				log.Print(string(stderr))
				http.Error(w, string(stderr), http.StatusInternalServerError)
				return
			}
			if len(stderr) != 0 {
				http.Error(w, string(stderr), http.StatusInternalServerError)
				return
			}

			f, err := os.Open(out)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer f.Close()
			http.ServeContent(w, r, "main.wasm", time.Now(), f)
			return
		}
	}

	http.ServeFile(w, r, path)
}

func main() {
	flag.Parse()
	//mime.AddExtensionType(".wasm", "application/wasm")
	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(*flagHTTP, nil))
}
