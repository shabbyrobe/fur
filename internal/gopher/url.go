package gopher

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type URL struct {
	Hostname string
	Port     int
	Root     bool
	ItemType ItemType
	Selector string
	Search   string
	Plus     string
}

// https://en.wikipedia.org/wiki/Gopher_(protocol)#URL_links
func (u URL) WWW() (url string, ok bool) {
	sel := u.Selector
	if u.ItemType == HTML && len(sel) >= 5 {
		if sel[0] == '/' {
			sel = sel[1:]
		}
		if (sel[0] == 'U' || sel[0] == 'u') &&
			(sel[1] == 'R' || sel[1] == 'r') &&
			(sel[2] == 'L' || sel[2] == 'l') &&
			sel[3] == ':' {
			return sel[4:], true
		}
	}

	return "", false
}

func (u URL) Host() string {
	p := u.Port
	if p == 0 {
		p = 70
	}
	return fmt.Sprintf("%s:%d", u.Hostname, p)
}

func (u URL) Query() string {
	if u.Search == "" {
		return u.Selector + "\r\n"
	} else {
		return u.Selector + "\t" + u.Search + "\r\n"
	}
}

func (u URL) String() string {
	var out strings.Builder
	out.WriteString("gopher://")
	out.WriteString(u.Hostname)

	if u.Port != 70 {
		out.WriteByte(':')
		out.WriteString(strconv.FormatInt(int64(u.Port), 10))
	}

	if !u.Root {
		out.WriteByte('/')
		out.WriteString(string(rune(u.ItemType)))
		out.WriteString(escape(u.Selector))

		n := 0
		if u.Plus != "" {
			n = 2
		} else if u.Search != "" {
			n = 1
		}

		switch n {
		case 2:
			out.WriteByte('\t')
			out.WriteString(escape(u.Search))
			out.WriteByte('\t')
			out.WriteString(escape(u.Plus))
		case 1:
			out.WriteByte('\t')
			out.WriteString(escape(u.Search))
		}
	}
	return out.String()
}

func (u URL) Parts() map[string]interface{} {
	// XXX: this is just here to make it easier to dump
	m := make(map[string]interface{}, 7)
	m["Hostname"] = u.Hostname
	m["Port"] = u.Port
	m["Root"] = u.Root
	m["ItemType"] = u.ItemType
	m["Selector"] = u.Selector
	m["Search"] = u.Search
	m["Plus"] = u.Plus
	return m
}

func IsWellKnownDummyHostname(s string) bool {
	s = strings.TrimSpace(s)

	// This is a collection of strings seen in real-world gopher servers
	// that indicate the hostname is a dummy:
	return s == "error.host" ||
		s == "fakeserver" ||
		s == "Error" ||
		s == "none" ||
		s == "fake" ||
		s == "(NULL)" ||
		s == "(FALSE)" ||
		s == "invalid" || // RFC2606 hostnames: https://tools.ietf.org/html/rfc2606
		s == "example" ||
		s == "." ||
		strings.HasSuffix(s, ".invalid") ||
		strings.HasSuffix(s, ".example")
}

func escape(s string) string {
	// XXX: currently wraps url.PathEscape(), which doesn't provide a clean way
	// _not_ to escape '/', so hack hack hack!
	return strings.Replace(url.PathEscape(s), "%2F", "/", -1)
}

// https://tools.ietf.org/html/rfc6335#section-5.1
var portEnd = regexp.MustCompile(`:([A-Za-z0-9-]+|\d+)$`)

func ParseURL(s string) (gu URL, err error) {
	u, err := url.Parse(s)
	if err != nil {
		return URL{}, err
	}

	// FIXME: interprets "localhost:7070" as "scheme:opaque"

	if u.Fragment != "" || u.Opaque != "" || u.User != nil {
		return URL{}, fmt.Errorf("gopher: invalid URL %q", u)
	}

	if u.Scheme == "" {
		u, err = url.Parse("gopher://" + u.String())
		if err != nil {
			return URL{}, err
		}
	} else if u.Scheme != "gopher" {
		return URL{}, fmt.Errorf("gopher: invalid URL %q", u)
	}

	h := u.Host
	if !portEnd.MatchString(h) {
		// SplitHostPort fails if there is no port with an error we can't catch:
		h += ":70"
	}

	var port string
	gu.Hostname, port, err = net.SplitHostPort(h)
	if err != nil {
		return URL{}, err
	}

	if port == "" {
		gu.Port = 70
	} else {
		portn, err := strconv.ParseInt(port, 0, 0)
		if err != nil {
			return URL{}, err
		}
		gu.Port = int(portn)
	}

	p := u.Path

	// FIXME: This will eat a bare '?' at the end of a selector, which may not be what we
	// want. At this point, I want to write a fully fledged URL parser even less (maybe
	// I'm a bit "edge-case"-d out after spending an evening with Gopher!). Perhaps later.
	if u.RawQuery != "" {
		p += "?" + u.RawQuery
	}

	plen := len(p)

	if plen > 0 && p[0] == '/' {
		p = p[1:]
		plen--
	}

	if plen == 0 {
		gu.Root = true

	} else {
		gu.ItemType = ItemType(p[0])
		p = p[1:]
		plen--
		s, field := 0, 0
		for i := 0; i <= plen; i++ {
			if i == plen || p[i] == '\t' {
				switch field {
				case 0:
					gu.Selector = p[s:i]
					field, s = field+1, i+1
				case 1:
					gu.Search = p[s:i]
					field, s = field+1, i+1
				case 2:
					gu.Plus = p[s:]
					goto pathDone
				}
			}
		}
	pathDone:
	}

	if err != nil {
		return gu, err
	}

	return gu, nil
}
