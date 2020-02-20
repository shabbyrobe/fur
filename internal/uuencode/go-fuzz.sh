#!/bin/bash
set -o errexit -o nounset -o pipefail

fuzz_file="$1-fuzz.zip" 
go generate -tags gofuzz

if [[ ! -e "$fuzz_file" ]]; then
    echo "unknown fuzz $1"
    exit 1
fi

go-fuzz -bin "$fuzz_file" -workdir testdata/"$1"

