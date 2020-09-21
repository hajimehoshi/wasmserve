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
	flagHTTP        = flag.String("http", ":8080", "HTTP bind address to serve")
	flagTags        = flag.String("tags", "", "Build tags")
	flagAllowOrigin = flag.String("allow-origin", "", "Allow specified origin (or * for all origins) to make requests to this server")
)

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
	cmd := exec.Command("go", "mod", "init", "example.com/m")
	cmd.Dir = path
	return cmd.CombinedOutput()
}

var (
	tmpWorkDir   = ""
	tmpOutputDir = ""
)

func ensureTmpWorkDir() (string, error) {
	if tmpWorkDir != "" {
		return tmpWorkDir, nil
	}

	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	tmpWorkDir = tmp
	return tmpWorkDir, nil
}

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
	fpath := filepath.Join(".", filepath.Base(upath))
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
	case "index.html", ".":
		if _, err := os.Stat(fpath); err != nil && !os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, "index.html", time.Now(), bytes.NewReader([]byte(indexHTML)))
		return
	case "wasm_exec.js":
		if _, err := os.Stat(fpath); err != nil && !os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f := filepath.Join(runtime.GOROOT(), "misc", "wasm", "wasm_exec.js")
		http.ServeFile(w, r, f)
		return
	case "main.wasm":
		if _, err := os.Stat(fpath); err != nil && !os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// go build
		args := []string{"build", "-o", filepath.Join(output, "main.wasm")}
		if *flagTags != "" {
			args = append(args, "-tags", *flagTags)
		}
		if len(flag.Args()) > 0 {
			args = append(args, flag.Args()[0])
		} else {
			args = append(args, ".")
		}
		log.Print("go ", strings.Join(args, " "))
		cmdBuild := exec.Command("go", args...)
		cmdBuild.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
		// If GO111MODULE is not specified explicilty, enable Go modules.
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

	http.ServeFile(w, r, filepath.Join(".", r.URL.Path))
}

func main() {
	flag.Parse()
	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(*flagHTTP, nil))
}
