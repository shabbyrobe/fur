package gopher

// TLSMode describes how the Client will respond in the presence of a URL
// with a gopher:// scheme. If the gophers:// scheme is used, the TLSMode
// is always "TLSInsist".
type TLSMode int

const (
	TLSModeDefault TLSMode = iota

	// The client will always attempt a TLS connection, and if it fails, an
	// error is returned.
	TLSInsist

	// The client will attempt a TLS connection, and if it fails, attempt
	// a plain-text connection.
	TLSWithInsecure

	// The client will not attempt a TLS connection; plain-text only.
	TLSDisabled
)

func (t TLSMode) downgrade() bool {
	return t == TLSWithInsecure
}

func (t TLSMode) resolve(force bool) TLSMode {
	if force {
		return TLSInsist
	}
	if t == TLSModeDefault {
		return TLSWithInsecure
	}
	return t
}

func (t TLSMode) shouldAttempt() bool {
	return t == TLSInsist || t == TLSWithInsecure
}
