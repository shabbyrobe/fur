package gopher

import "time"

type Caps interface {
	Version() int
	ExpiresAfter() time.Duration

	PathConfig() (*PathConfig, error)
	ServerInfo() (*ServerInfo, error)
	Software() (name, version string)

	// Default text encoding for content types 0 and 1.
	// If this returns an empty string, UTF-8 is presumed.
	DefaultEncoding() string
}

var UnixPathConfig = PathConfig{
	Delimiter:        "/",
	Identity:         ".",
	Parent:           "..",
	ParentDouble:     false,
	EscapeCharacter:  '\\',
	KeepPreDelimiter: false,
}

type PathConfig struct {
	// Refers to how the server separates folders from each other; Unix machines use `/`,
	// Microsoft machines use `\`, and obsolete Macs use `:`
	Delimiter string

	// Refers to the shorthand used by an operating system to mean "this directory"; UNIX
	// machines use `.`.
	Identity string

	// Refers to the shorthand for "the directory immediately above", and is `..` on UNIX
	// and Microsoft systems.
	Parent string

	// Refers to an oddball feature of obsolete Macs: two consecutive path delimiters are
	// used to refer to the parent directory. For all systems other than pre-OS X
	// Macintoshes, this should be false.
	ParentDouble bool

	// Tells the client the escape character for quoting delimiters when they appear in
	// selectors; most of the time, this is `\\`.
	EscapeCharacter byte

	// Tells the client not to cut everything up to the first path delimiter; most of the
	// time, this should be `FALSE`.
	KeepPreDelimiter bool
}

type ServerInfo struct {
	Software     string
	Version      string
	Architecture string
	Description  string
	Geolocation  string
	AdminEmail   string
}
