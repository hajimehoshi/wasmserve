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

// +build example

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall/js"
)

func main() {
	flag.Parse()
	fmt.Println(flag.Args())

	p := js.Global().Get("document").Call("createElement", "p")
	p.Set("innerText", strings.Join([]string{
		"Hello, World!",
		fmt.Sprintf("args=%q", os.Args),
		fmt.Sprintf("env=%q", os.Environ()),
	}, "\n"))
	js.Global().Get("document").Get("body").Call("appendChild", p)
}
