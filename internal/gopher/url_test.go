package gopher

import "testing"

func TestURLIPv6(t *testing.T) {
	u, err := ParseURL("gopher://[::]:70/1/yep")
	if err != nil {
		t.Fatal(err)
	}

	if u.Hostname != "::" {
		t.Fatal(err)
	}
	if u.Port != "70" {
		t.Fatal(err)
	}
	if u.String() != "gopher://[::]/1/yep" {
		t.Fatal(u.String())
	}
}
