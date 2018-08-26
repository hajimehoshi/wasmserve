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
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
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
  const result = await WebAssembly.instantiate(src, go.importObject);
  go.run(result.instance);
})();
</script>
`

var (
	flagHTTP = flag.String("http", ":8080", "HTTP bind address to serve")
	flagTags = flag.String("tags", "", "Build tags")
)

func gobin() string {
	return filepath.Join(runtime.GOROOT(), "bin", "go")
}

func ensureModule(path string) ([]byte, error) {
	_, err := os.Stat(filepath.Join(path, "go.mod"))
	if err == nil {
		return nil, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	log.Print("(", path, ")")
	log.Print("go mod init example.com/m")
	cmd := exec.Command(gobin(), "mod", "init", "example.com/m")
	cmd.Dir = path
	return cmd.CombinedOutput()
}

var tmpDir = ""

func ensureTmp() (string, error) {
	if tmpDir != "" {
		return tmpDir, nil
	}

	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	tmpDir = tmp
	return tmpDir, nil
}

func handle(w http.ResponseWriter, r *http.Request) {
	tmp, err := ensureTmp()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if stderr, err := ensureModule(tmp); err != nil {
		log.Print(err)
		log.Print(string(stderr))
		http.Error(w, string(stderr), http.StatusInternalServerError)
		return
	}

	upath := r.URL.Path[1:]
	cfg := &packages.Config{
		Dir: tmp,
		Env: append(os.Environ(), "GO111MODULE=on", "GOOS=js", "GOARCH=wasm"),
	}
	if tags := *flagTags; tags != "" {
		cfg.BuildFlags = []string{"-tags", tags}
	}
	pkg := filepath.Dir(upath)
	pkgs, err := packages.Load(cfg, pkg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fpath := filepath.Join(filepath.Dir(pkgs[0].GoFiles[0]), filepath.Base(upath))

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

	if strings.HasSuffix(r.URL.Path, "/") {
		fpath = filepath.Join(fpath, "index.html")
	}

	switch filepath.Base(fpath) {
	case "index.html":
		if _, err := os.Stat(fpath); os.IsNotExist(err) {
			http.ServeContent(w, r, "index.html", time.Now(), bytes.NewReader([]byte(indexHTML)))
			return
		}
	case "wasm_exec.js":
		if _, err := os.Stat(fpath); os.IsNotExist(err) {
			f := filepath.Join(runtime.GOROOT(), "misc", "wasm", "wasm_exec.js")
			http.ServeFile(w, r, f)
			return
		}
	case "main.wasm":
		if _, err := os.Stat(fpath); os.IsNotExist(err) {
			// go build
			args := []string{"build", "-o", "main.wasm"}
			if *flagTags != "" {
				args = append(args, "-tags", *flagTags)
			}
			args = append(args, pkg)
			log.Print("go ", strings.Join(args, " "))
			cmdBuild := exec.Command(gobin(), args...)
			cmdBuild.Env = append(os.Environ(), "GO111MODULE=on", "GOOS=js", "GOARCH=wasm")
			cmdBuild.Dir = tmp
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

			f, err := os.Open(filepath.Join(tmp, "main.wasm"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer f.Close()
			http.ServeContent(w, r, "main.wasm", time.Now(), f)
			return
		}
	}

	http.ServeFile(w, r, fpath)
}

func main() {
	flag.Parse()
	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(*flagHTTP, nil))
}
