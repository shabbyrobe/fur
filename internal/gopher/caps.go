package gopher

import "time"

var (
	magic = []byte("CAPS")
)

type Caps struct {
	Version int
	Expiry  time.Duration
	Entries []Cap
}

type Cap interface{}

type CapKeyValue struct {
	Key   string
	Value string
	Line  int
}

var _ Cap = CapKeyValue{}

type CapComment struct {
	Comment string
	Line    int
}

var _ Cap = CapComment{}
