package main

import (
	"testing"

	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/cmdytest"
)

func TestCommand(t *testing.T) {
	tester := cmdytest.ExampleTester{
		TestName: "fur",
		Builder:  func() cmdy.Command { return &command{} },
	}
	tester.TestExamples(t)

	nonDocExamples := cmdy.Examples{
		// FIXME: localhost without scheme should work just fine, but it doesn't just yet:
		cmdy.Example{Command: "localhost:8080", Code: cmdy.ExitUsage},
	}
	for _, ex := range nonDocExamples {
		tester.TestExample(t, ex)
	}
}
