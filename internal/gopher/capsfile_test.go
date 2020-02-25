package gopher

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestParseCapsSeparateComments(t *testing.T) {
	cf := strings.Join([]string{
		`CAPS`,
		`# foo`,
		``,
		`# bar`,
	}, "\n")

	caps, err := ParseCapsBytes("file", []byte(cf), 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = caps
	// spew.Dump(caps)
}

func TestParseCapsInvalidKV(t *testing.T) {
	for idx, tc := range []struct {
		in string
	}{
		{"foo"},
		{"$foo"},
		{"$foo yep"},
		{"="},
		{"=1"},
		{"$="},
		{"$=1"},
	} {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			_, err := ParseCapsBytes("file", []byte("CAPS\n"+tc.in), 0)
			if !errors.Is(err, ErrCapsKeyValueInvalid) {
				t.Fatal()
			}
		})
	}
}
