package gopher

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

var (
	tokCapsMagic           = []byte("CAPS")
	ErrCapsKeyValueInvalid = errors.New("caps: invalid key=value")
)

type CapsFile struct {
	Name         string
	Entries      []CapEntry
	keyIndex     map[string]*CapKeyValue
	version      int
	expiresAfter time.Duration
}

var _ Caps = &CapsFile{}

func NewCapsFile(name string) *CapsFile {
	return &CapsFile{
		Name:     name,
		Entries:  make([]CapEntry, 0, 32),
		keyIndex: map[string]*CapKeyValue{},
	}
}

func (cf *CapsFile) Version() int { return cf.version }

func (cf *CapsFile) ExpiresAfter() time.Duration { return cf.expiresAfter }

func (cf *CapsFile) String(key string) (s string, ok bool) {
	kv := cf.keyIndex[strings.ToLower(key)]
	if kv == nil {
		return "", false
	}
	return kv.Value, true
}

func (cf *CapsFile) Bool(key string) (v bool, ok bool, err error) {
	kv := cf.keyIndex[strings.ToLower(key)]
	if kv == nil {
		return false, false, nil
	}
	v, err = strconv.ParseBool(kv.Value)
	return v, true, err
}

func (cf *CapsFile) Int64(key string) (v int64, ok bool, err error) {
	kv := cf.keyIndex[key]
	if kv == nil {
		return 0, false, nil
	}
	v, err = strconv.ParseInt(key, 10, 64)
	return v, true, err
}

func (cf *CapsFile) Software() (name, version string) {
	name, _ = cf.String("ServerSoftware")
	version, _ = cf.String("ServerVersion")
	return name, version
}

func (cf *CapsFile) ServerInfo() (*ServerInfo, error) {
	var si ServerInfo

	si.Software, _ = cf.String("ServerSoftware")
	si.Version, _ = cf.String("ServerVersion")
	si.Architecture, _ = cf.String("ServerArchitecture")
	si.Description, _ = cf.String("ServerDescription")
	si.Geolocation, _ = cf.String("ServerGeolocationString")
	si.AdminEmail, _ = cf.String("ServerAdmin")

	return &si, nil
}

func (cf *CapsFile) DefaultEncoding() string {
	enc, _ := cf.String("DefaultEncoding")
	return enc
}

func (cf *CapsFile) PathConfig() (*PathConfig, error) {
	pc := UnixPathConfig

	var errs []string

	// XXX: The caps key is 'PathDelimeter', which is a real-world common-use misspelling
	// a-la 'HTTP Referer':
	if d, ok := cf.String("PathDelimeter"); ok {
		pc.Delimiter = d
	}
	if d, ok := cf.String("PathIdentity"); ok {
		pc.Identity = d
	}
	if d, ok := cf.String("PathParent"); ok {
		pc.Parent = d
	}

	if b, ok, err := cf.Bool("PathParentDouble"); ok {
		pc.ParentDouble = b
	} else if err != nil {
		errs = append(errs, fmt.Sprintf("PathParentDouble value invalid: %s", err))
	}

	if d, ok := cf.String("PathEscapeCharacter"); ok {
		if len(d) != 1 {
			errs = append(errs, fmt.Sprintf("PathEscapeCharacter %q invalid, must be 1 character", d))
		} else {
			pc.EscapeCharacter = d[0]
		}
	}

	if b, ok, err := cf.Bool("PathKeepPreDelimiter"); ok {
		pc.ParentDouble = b
	} else if err != nil {
		errs = append(errs, fmt.Sprintf("PathKeepPreDelimiter value invalid: %s", err))
	}

	if len(errs) > 0 {
		return &pc, fmt.Errorf("gopher: caps path config invalid: %s", strings.Join(errs, ", "))
	}

	return &pc, nil
}

type CapEntry interface {
	capEntry()
}

type CapKeyValue struct {
	Key   string
	Value string
	Raw   []byte
}

func (*CapKeyValue) capEntry() {}

type CapComment []byte

func (CapComment) capEntry() {}

type CapWsp []byte

func (CapWsp) capEntry() {}

type CapDot []byte

func (CapDot) capEntry() {}

func ParseCaps(name string, rdr io.Reader, flag ParseCapsFlag) (*CapsFile, error) {
	const maxCapsFileLine = 2048
	const maxCapsSize = 1 << 17

	data, err := readAtMost(rdr, maxCapsSize)
	if err != nil {
		return nil, err
	}

	return ParseCapsBytes(name, data, flag)
}

type ParseCapsFlag int

const (
	// Dodgy gopher servers can and do stuff '.\r\n' lines in to the caps file. We allow
	// it by default, but if this flag is set, we forbid it.
	CapsForbidDot ParseCapsFlag = 1 << iota
)

func ParseCapsBytes(name string, data []byte, flag ParseCapsFlag) (*CapsFile, error) {
	const (
		lineComment = iota + 1
		lineKV
		lineWsp
		lineDot // Malformed servers could send these at any time
		lineEOF
		lineInvalid
	)

	if !bytes.HasPrefix(data, tokCapsMagic) {
		return nil, fmt.Errorf("gopher: missing caps magic")
	}

	file := NewCapsFile(name)

	var pos, sz = len(tokCapsMagic), len(data)
	var start = pos
	var linetypLast int
	var lnum = 1

	for pos <= sz {
		var line []byte
		var end int
		var linetyp int

		if pos < sz {
			nl := bytes.IndexByte(data[pos:], '\n')
			if nl >= 0 {
				end = nl + 1
				line = dropCR(data[pos:pos+nl], nl)
			} else {
				end = 0
				line = data[pos:]
			}

			if len(line) == 0 || line[0] == ' ' || line[0] == '\t' {
				linetyp = lineWsp
			} else if line[0] == '.' {
				linetyp = lineDot
			} else if line[0] == '#' {
				linetyp = lineComment
			} else {
				linetyp = lineKV
			}

		} else {
			linetyp = lineEOF
		}

		if linetypLast > 0 && linetypLast != linetyp {
			switch linetypLast {
			case lineDot:
				file.Entries = append(file.Entries, CapDot(data[start:pos]))
			case lineWsp:
				file.Entries = append(file.Entries, CapWsp(data[start:pos]))
			case lineComment:
				file.Entries = append(file.Entries, CapComment(data[start:pos]))
			}
			start = pos
		}

		if linetyp == lineEOF {
			linetypLast = lineEOF
			break
		}

		switch linetyp {
		case lineKV:
			k, v, err := capsParseKV(line)
			if err != nil {
				return file, fmt.Errorf("gopher: caps file error at line %d: %w", lnum, err)
			}
			kv := CapKeyValue{Key: k, Value: v, Raw: data[start : pos+end]}
			file.Entries = append(file.Entries, &kv)
			file.keyIndex[strings.ToLower(k)] = &kv
			start = pos + end
			linetypLast = 0

			switch k {
			case "CapsFileVersion":
				iv, err := strconv.ParseInt(v, 10, 0)
				if err != nil {
					return nil, err
				}
				file.version = int(iv)

			case "ExpireCapsAfter":
				iv, err := strconv.ParseInt(v, 10, 32) // 32-bit to prevent overflow
				if err != nil {
					return nil, err
				}
				file.expiresAfter = time.Duration(iv) * time.Second
			}

		case lineDot:
			if len(line) != 1 || flag&CapsForbidDot != 0 {
				return file, fmt.Errorf("gopher: caps file error at line %d: invalid key", lnum)
			}
			linetypLast = linetyp

		default:
			linetypLast = linetyp
		}

		pos += end
		lnum++
	}

	if linetypLast != lineEOF {
		return file, fmt.Errorf("gopher: premature end of caps file")
	}

	return file, nil
}

func capsParseKV(line []byte) (k, v string, err error) {
	index := bytes.IndexByte(line, '=')
	if index < 0 {
		return k, v, ErrCapsKeyValueInvalid
	}

	// GopherII spec:
	// "Any amount of whitespace (spaces and tabs) around the equals sign is acceptable."
	k = strings.TrimRight(string(line[:index]), " \t")
	if len(k) == 0 {
		return k, v, ErrCapsKeyValueInvalid
	}

	for _, b := range k {
		if !((b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')) {
			return k, v, ErrCapsKeyValueInvalid
		}
	}

	v = strings.TrimLeft(string(line[index+1:]), " \t")

	return k, v, nil
}

func dropCR(data []byte, sz int) []byte {
	if len(data) > 0 && data[sz-1] == '\r' {
		return data[0 : sz-1]
	}
	return data
}

func readAtMost(r io.Reader, limit int64) (bts []byte, err error) {
	limRdr := &io.LimitedReader{R: r, N: limit}
	bts, err = ioutil.ReadAll(limRdr)
	if err != nil {
		return bts, err
	}
	if limRdr.N <= 0 {
		return bts, fmt.Errorf("gopher: caps too large")
	}
	return bts, nil
}
