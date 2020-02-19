package main

import (
	"context"
	"os"

	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/cmdyutil"
)

func main() {
	if err := run(); err != nil {
		cmdy.Fatal(err)
	}
}

func run() error {
	bld := func() cmdy.Command { return &command{} }
	return cmdyutil.InterruptibleRun(context.Background(), os.Args[1:], bld)
}
