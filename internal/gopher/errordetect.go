package gopher

import (
	"bytes"
	"regexp"
)

var (
	// https://tools.ietf.org/html/draft-matavka-gopher-ii-02#section-9
	tokPlusError     = []byte{'-', '-'}
	nilReadCloserVal = nilReadCloser{}
)

type ErrFactory func(status Status, msg string, confidence float64) *Error

func DetectError(data []byte, errFactory ErrFactory) *Error {
	const firstLineMax = 200

	dlen := len(data)

	// Unless the client considers the item type we are requesting to be a binary
	// selector, an empty response is an error:
	if dlen == 0 {
		// FIXME: return "Empty response error"
		return errFactory(StatusEmpty, "", 1)
	}

	// If the first line is crazy long, we're probably on the wrong track:
	firstNl := bytes.IndexByte(data, '\n')
	if firstNl > firstLineMax || (firstNl < 0 && dlen > firstLineMax) {
		return nil
	}
	if firstNl < 0 {
		firstNl = dlen
	}

	// Try to detect error responses that start with '--'. These probably won't happen
	// if we didn't issue a Gopher+/GopherII request directly:
	// https://tools.ietf.org/html/draft-matavka-gopher-ii-02#section-9
	if bytes.HasPrefix(data, tokPlusError) {
		status, msg, found := extractGopherIIError(data)
		if found {
			return errFactory(status, msg, 1)
		} else {
			return nil
		}
	}

	// Step 2: Try to detect error responses that contain a single directory entry of type
	// '3', which may be preceded and/or followed by a list of 'i' lines.
	if (data[0] == 'i' || data[0] == '3') && firstNl > 0 {
		status, msg, found := extractDirentError(data)
		if found {
			return errFactory(status, msg, 0.9)
		} else {
			return nil
		}
	}

	// Starting to get more loose; check if data starts with well-known error prefix,
	// then check against some more complex patterns if we have a match:
	check := errorTrimRightWsp(data, dlen)
	found, n := errorPrefixMatcher.Find(check)
	if found {
		confidence := 0.4
		if errorPatternLoose.Match(check[n:]) {
			confidence = 0.7
		}
		return errFactory(StatusGeneralError, string(data[:firstNl]), confidence)
	}

	return nil
}

func extractGopherIIError(data []byte) (status Status, msg string, found bool) {
	const (
		stateHyphen1 = iota
		stateHyphen2
		stateStatus
		stateMessage
		stateEndLF
	)

	var state int
	var msgStart int

	for idx, c := range data {
		switch state {
		case stateHyphen1:
			if c == '-' {
				state = stateHyphen2
			} else {
				return 0, "", false
			}

		case stateHyphen2:
			if c == '-' {
				state = stateStatus
			} else {
				return 0, "", false
			}

		case stateStatus:
			if c >= '0' || c <= '9' {
				status = status*10 + Status(c) - '0'
				msgStart = idx + 1
			} else if c == '\t' {
				state = stateMessage
			} else {
				return 0, "", false
			}

		case stateMessage:
			if c == '\r' {
				state = stateEndLF
				msg = string(data[msgStart:idx])
			} else if c == '\n' {
				return 0, "", false
			}

		case stateEndLF:
			if c == '\n' {
				return status, msg, true
			}
			return 0, "", false
		}
	}

	return 0, "", false
}

func extractDirentError(data []byte) (status Status, msg string, found bool) {
	dsz := len(data)

	var dirent Dirent
	var lnum = 1

	for idx, start := 0, 0; idx >= 0 && start < dsz; lnum++ {
		idx = bytes.IndexByte(data[start:], '\n')
		var line []byte
		if idx < 0 {
			line = data[start:]
		} else {
			line = data[start : start+idx]
		}

		start += idx + 1

		if line[0] == '3' {
			if found {
				return 0, "", false
			}

			line = errorTrimRightCRLF(line, len(line))
			if err := unmarshalDirent(string(line), lnum, &dirent); err != nil {
				// XXX: if this is the last line in 'data', we may be trying to
				// read a truncated dirent, so if we have already found a valid
				// dirent, let's not overwrite it
				break
			}
			found = true
		}
	}

	if found {
		// XXX: We can try more string matching tricks to get a better code here?
		// - "Malformed request"
		// - "Happy helping ☃ here: Sorry, your selector does not start with / or contains '..'. That's illegal here."
		// - "Sorry, but the requested token '/caps.txt' could not be found."
		// - " '/robots.txt' doesn't exist!"
		// - "'/caps.txt' does not exist (no handler found)"
		return StatusGeneralError, dirent.Display, true
	}

	return 0, "", false
}

var (
	errorPrefixMatcher = errorMatcherBuild([][]byte{
		[]byte("an error occurred:"),
		[]byte("error:"),
		[]byte("file:"),
	})

	errorPatternLoose = regexp.MustCompile(`` +
		`(?i)` +
		`(` +
		`\bnot found\b` +
		`|` +
		`resource .*? does not exist` +
		`)` +
		``)
)

func errorMatcherBuild(msgs [][]byte) *errorMatcherNode {
	node := &errorMatcherNode{}
	for _, msg := range msgs {
		cur := node
		for _, b := range msg {
			if cur.next[b] == nil {
				cur.next[b] = &errorMatcherNode{}
			}
			cur = cur.next[b]
		}
		cur.match = true
	}
	return node
}

type errorMatcherNode struct {
	next  [256]*errorMatcherNode
	match bool
}

func (node *errorMatcherNode) Find(buf []byte) (found bool, n int) {
	cur := node
	for idx, b := range buf {
		b = caseFold[b]
		if cur.next[b] == nil {
			break
		}
		cur = cur.next[b]
		if cur.match {
			found = true
			n = idx
		}
	}

	return found, n
}

var caseFold = [256]byte{}

func init() {
	for i := 0; i < 256; i++ {
		b := byte(i)
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		caseFold[i] = b
	}
}

// errorTrimRight trims bytes, rather than runes, off a byte slice
func errorTrimRightWsp(in []byte, sz int) []byte {
	if sz == 0 {
		return in
	}

	end := sz
	for i := sz - 1; i >= 0; i-- {
		if in[i] == ' ' || in[i] == '\n' || in[i] == '\r' || in[i] == '\t' {
			end--
		} else {
			break
		}
	}
	return in[:end]
}

// errorTrimRightCRLF trims bytes, rather than runes, off a byte slice
func errorTrimRightCRLF(in []byte, sz int) []byte {
	if sz == 0 {
		return in
	}

	end := sz
	for i := sz - 1; i >= 0; i-- {
		if in[i] == '\n' || in[i] == '\r' {
			end--
		} else {
			break
		}
	}
	return in[:end]
}
