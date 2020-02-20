#!/bin/sh
# Don't run this file directly. It's executed as part of TestGopherJSCanBeModules.

set -e

tmp=$(mktemp -d "${TMPDIR:-/tmp}/gopherjsmodules_test.XXXXXXXXXX")

cleanup() {
    rm -rf "$tmp"
    exit
}

trap cleanup EXIT HUP INT TERM

# Make a hello project that will Go modules GopherJS.
mkdir -p "$tmp/src/example.org/hello"
echo 'package main

import (
    "github.com/gopherjs/gopherjs/js"
    _ "github.com/gopherjs/websocket"
)

func main() {
    js.Global.Get("console").Call("log", "hello using js pkg")
}' > "$tmp/src/example.org/hello/main.go"

# install gopherjs
go install github.com/gopherjs/gopherjs

# go mod init
(cd "$tmp/src/example.org/hello" && go mod init example.org/hello)

# go mod tidy
(cd "$tmp/src/example.org/hello" && go mod tidy)

# Use it to build and run the hello command.
(cd "$tmp/src/example.org/hello" && "$HOME/go/bin/gopherjs" run main.go)
