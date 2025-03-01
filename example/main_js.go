// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2018 Hajime Hoshi

package main

import (
	"flag"
	"fmt"
	"os"
	"syscall/js"
)

func main() {
	flag.Parse()
	fmt.Println(flag.Args())

	p := js.Global().Get("document").Call("createElement", "p")
	p.Set("innerText", "Hello, World!")
	js.Global().Get("document").Get("body").Call("appendChild", p)

	pre := js.Global().Get("document").Call("createElement", "pre")
	pre.Set("innerText", fmt.Sprintf("args: %q\nevn: %q", os.Args, os.Environ()))
	pre.Get("style").Set("whiteSpace", "pre-wrap")
	js.Global().Get("document").Get("body").Call("appendChild", pre)
}
