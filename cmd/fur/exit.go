package main

import "github.com/shabbyrobe/furlib/gopher"

// Just pulled whatever the closest match is out of sysexits.h, for better or worse...
// It'll do until some greybeard comes and yells at me for "holding it wrong".
//
// https://www.freebsd.org/cgi/man.cgi?query=sysexits&apropos=0&sektion=0&manpath=FreeBSD+4.3-RELEASE&format=html
// https://stackoverflow.com/questions/1101957/are-there-any-standard-exit-status-codes-in-linux
var (
	exitCodes = map[gopher.Status]byte{
		gopher.OK:                   0,
		gopher.StatusBadRequest:     65, // EX_DATAERR
		gopher.StatusUnauthorized:   77, // EX_NOPERM
		gopher.StatusForbidden:      77, // EX_NOPERM
		gopher.StatusNotFound:       69, // EX_UNAVAILABLE
		gopher.StatusRequestTimeout: 74, // EX_IOERR
		gopher.StatusGone:           69, // EX_UNAVAILABLE
		gopher.StatusInternal:       70, // EX_SOFTWARE
		gopher.StatusNotImplemented: 65, // EX_DATAERR
		gopher.StatusUnavailable:    69, // EX_UNAVAILABLE

		gopher.StatusGeneralError: 65, // EX_DATAERR
		gopher.StatusEmpty:        65, // EX_DATAERR
	}
)

func exitCode(status gopher.Status, dflt int) int {
	out, ok := exitCodes[status]
	if !ok {
		return dflt
	}
	return int(out)
}
