package gopher

import (
	"fmt"
	"testing"
)

func TestPopulateRequestURL(t *testing.T) {
	const withData, noData = true, false

	// func populateRequestURL(url *URL, line []byte) (hasData bool, err error) {
	for idx, tc := range []struct {
		line string
		out  string
		data bool
	}{
		{"", "gopher://invalid", noData},
		{"foo", "gopher://invalid/0foo", noData},
		{"foo\tsearch", "gopher://invalid/0foo%09search", noData},
		{"foo\tsearch\t1", "gopher://invalid/0foo%09search", withData},
		{"foo\t\t1", "gopher://invalid/0foo", withData},
	} {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			var u = URL{Hostname: "invalid"}
			hasData, err := populateRequestURL(&u, []byte(tc.line))
			if err != nil {
				t.Fatal(err)
			}
			if hasData != tc.data {
				t.Fatal(err)
			}
			if u.String() != tc.out {
				t.Fatal(u.String(), "!=", tc.out)
			}
		})
	}
}
