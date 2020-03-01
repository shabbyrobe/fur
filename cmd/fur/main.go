package main

import (
	"context"
	"os"

	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/cmdyutil"
	"github.com/shabbyrobe/golib/profiletools"
)

func main() {
	if err := run(); err != nil {
		cmdy.Fatal(err)
	}
}

func run() error {
	// FIXME: remove profiling
	pt := profiletools.EnvProfile("FUR_")
	defer pt.Stop()

	bld := func() cmdy.Command { return &command{} }
	return cmdyutil.InterruptibleRun(context.Background(), os.Args[1:], bld)
}
