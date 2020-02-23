package gopher

import (
	"fmt"
	"testing"
)

func TestErrorDetect(t *testing.T) {
	for idx, tc := range []struct {
		in string
	}{
		{`3'/caps.txt' does not exist (no handler found)		error.host	1`},
		{"3 '/caps.txt' doesn't exist!		error.host	1"},
		{"3`caps.txt' invalid.	Error	Error	0"},
		{`3'caps.txt' No such file or directory (2)`},
		{`3caps.txt NOT FOUND`},
		{`3Error accessing /caps.txt.		error.host	1`},
		{`3 File not found`},
		{`3file not found	fake	(NULL)	0`},
		{`3Happy helping ☃ here: Sorry, your selector contains '..'. That's illegal here.	Err	localhost	70`},
		{`3Happy helping ☃ here: Sorry, your selector does not start with / or contains '..'. That's illegal here.	Err	localhost	70`},
		{`3Malformed request	fakeselector	fakeserver	70`},
		{`3not found	(NULL)	error.host	0`},
		{`3open path/to/caps.txt: no such file or directory		error.host	1`},
		{`3Sorry, but the requested token 'caps.txt' could not be found.	Err	localhost	70`},
		{`3Sorry! I could not find caps.txt`},
		{`3The provided selector is invalid.		example.com	70`},
		{`3"/usr/stevie/gophercaps.txt" not found	error.file	error.host	0`},
		{`An error occurred: Resource not found.`},
		{`Error: 404 Not Found`},
		{`Error: File or directory not found!`},
		{`Error: Page not found	example.com	70`},
		{`Error: resource caps.txt does not exist on example.com`},
		{`File: '/caps.txt' not found.`},
		{`Error: 404 Not Found\n\nThe requested URL was not found on the server. If you entered the URL\nmanually please check your spelling and try again.`},
		{`` +
			`i   ____            _       ____      _ _` + "\n" +
			`i  |  _ \ _   _ ___| |_ ___|  _ \  __| | | __` + "\n" +
			"i  | | | | | | / __| __/ _ | | | |/ _` | |/ /" + "\n" +
			`i  | |_| | |_| \__ \ ||  __/ |_| | (_| |   < ` + "\n" +
			`i  |____/ \__,_|___/\__\___|____(_)__,_|_|\_\` + "\n" +
			`i                    - a strange place indeed` + "\n" +
			`i ` + "\n" +
			`i ` + "\n" +
			`3Sorry! I could not find caps.txt`,
		},
	} {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			fn := func(status Status, msg string, confidence float64) *Error {
				return NewError(URL{}, status, msg, confidence)
			}
			if DetectError([]byte(tc.in), fn) == nil {
				t.Fatal(tc.in)
			}
		})
	}
}
