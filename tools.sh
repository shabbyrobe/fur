#!/bin/bash

cmd-install() {
    GO111MODULE=on go install -trimpath -ldflags "-s -w" "$@" ./cmd/fur
}

cmd-build() {
    GO111MODULE=on go build -trimpath -ldflags "-s -w" "$@" ./cmd/fur
}

cmd-sloc() {
    tokei --exclude '*_test.go' .
}

cmd-binsz() {
    cmd-install
    ls -la "$( command -v fur )"
}

"cmd-$1" "${@:2}"

