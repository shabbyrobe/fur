package gopher

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// URL implements most of the Gopher URL scheme (excluding the crazy
// Gopher+ stuff).
//
// http://tools.ietf.org/html/rfc4266
// https://www.w3.org/Addressing/URL/4_1_Gopher+.html
type URL struct {
	Secure   bool
	Hostname string
	Port     string
	Root     bool
	ItemType ItemType
	Selector string
	Search   string
}

func (u URL) AsMetaItem(records ...string) URL {
	if len(records) == 0 {
		u.Search = string(MetaItem)
	} else {
		u.Search = recordSearch(MetaItem, records...)
	}
	return u
}

func (u URL) AsMetaDir(records ...string) URL {
	if len(records) == 0 {
		u.Search = string(MetaDir)
	} else {
		u.Search = recordSearch(MetaDir, records...)
	}
	return u
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

func (u URL) IsMeta() bool {
	// A GopherIIbis client may request the metadata for a specific selector
	// by sending a string in the following form:
	//	<selector>^I![CR][LF]
	//
	// It is possible to retrieve metadata for an *entire directory*. The `INFO` record
	// serves to separate metadata for one file from metadata for another. For example:
	//	<selector>^I&[CR][LF]
	//
	return len(u.Search) > 0 && (u.Search[0] == '!' || u.Search[0] == '&')
}

func (u URL) MetaType() MetaType {
	if len(u.Search) > 0 {
		c := u.Search[0]
		switch c {
		case '!', '&':
			return MetaType(c)
		}
	}
	return MetaNone
}

func (u URL) CanFetch() bool {
	return u.ItemType.CanFetch() && !IsWellKnownDummyHostname(u.Hostname)
}

func (u URL) Host() string {
	p := u.Port
	if p == "" {
		p = "70"
	}
	return net.JoinHostPort(u.Hostname, p)
}

func (u URL) URL() URL { return u }

func (u URL) String() string {
	var out strings.Builder
	if u.Secure {
		out.WriteString("gophers://")
	} else {
		out.WriteString("gopher://")
	}

	if strings.IndexByte(u.Hostname, ':') >= 0 {
		out.WriteByte('[')
		out.WriteString(u.Hostname)
		out.WriteByte(']')
	} else {
		out.WriteString(u.Hostname)
	}

	if u.Port != "70" {
		out.WriteByte(':')
		out.WriteString(u.Port)
	}

	if !u.Root {
		out.WriteByte('/')
		out.WriteString(string(rune(u.ItemType)))
		out.WriteString(escape(u.Selector))

		if u.Search != "" {
			out.WriteString("%09")
			out.WriteString(escape(u.Search))
		}
	}
	return out.String()
}

func (u URL) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *URL) UnmarshalText(b []byte) error {
	v, err := ParseURL(string(b))
	if err != nil {
		return err
	}
	*u = v
	return nil
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
	return m
}

func IsWellKnownDummyHostname(s string) bool {
	s = strings.TrimSpace(s)

	// This is a collection of strings seen in real-world gopher servers
	// that indicate the hostname is a dummy:
	return s == "error.host" ||
		s == "error" ||
		s == "fake" ||
		s == "fakeserver" ||
		s == "none" ||
		s == "invalid" || // RFC2606 hostnames: https://tools.ietf.org/html/rfc2606
		s == "example" ||
		s == "." ||
		s == "(null)" ||
		s == "(false)" ||
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
	} else {
		switch u.Scheme {
		case "gopher":
		case "gophers":
			gu.Secure = true
		default:
			return URL{}, fmt.Errorf("gopher: invalid URL %q", u)
		}
	}

	h := u.Host
	if !portEnd.MatchString(h) {
		// SplitHostPort fails if there is no port with an error we can't catch.
		// XXX: This also presumes we are using the "Lohmann Model" for TLS, rather
		// than a distinct port. Probably worth asking on the mailing list.
		h += ":70"
	}

	gu.Hostname, gu.Port, err = net.SplitHostPort(h)
	if err != nil {
		return URL{}, err
	}
	if gu.Port == "" {
		gu.Port = "70"
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

func recordSearch(meta MetaType, records ...string) string {
	var sb strings.Builder
	sb.WriteByte(byte(meta))
	for _, rec := range records {
		if len(rec) == 0 {
			continue
		}
		if rec[0] != '+' {
			sb.WriteByte('+')
		}
		sb.WriteString(rec)
	}
	return sb.String()
}
