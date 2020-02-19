package gopher

import "strconv"

type ItemType byte

const (
	File       ItemType = '0'
	Dir        ItemType = '1'
	CSOServer  ItemType = '2' // https://en.wikipedia.org/wiki/CCSO_Nameserver
	ItemError  ItemType = '3'
	BinHex     ItemType = '4' // Ancient pre OS X Mac format
	DOSArchive ItemType = '5' // Client must read until the TCP connection closes. Beware.
	UUEncoded  ItemType = '6'
	Search     ItemType = '7'
	Telnet     ItemType = '8' // Connect to given host at given port. The name to login as at this host is in the selector string.
	Binary     ItemType = '9' // Client must read until the TCP connection closes. Beware.

	// The information applies to a duplicated server. The information contained within is
	// a duplicate of the primary server. The primary server is defined as the last
	// DirEntity that is has a non-plus "Type" field. The client should use the
	// transaction as defined by the primary server Type field.
	Duplicate ItemType = '+'

	GIF   ItemType = 'g'
	Image ItemType = 'I' // Item is some kind of image file. Client gets to decide.

	// The information applies to a tn3270 based telnet session. Connect to given host at
	// given port. The name to login as at this host is in the selector string.
	TN3270 ItemType = 'T'

	// Non-canonical:
	Doc   = 'D'
	HTML  = 'h'
	Info  = 'i'
	Sound = 's'
)

var itemTypeStrings [256]string

func init() {
	for i := 0; i < 256; i++ {
		b := byte(i)
		itemTypeStrings[b] = strconv.QuoteRune(rune(b))
	}
}

func (i ItemType) String() string {
	return itemTypeStrings[i]
}

func (i ItemType) CanFetch() bool {
	return i != Duplicate && i != Telnet && i != TN3270 && i != CSOServer
}

func (i ItemType) IsSearch() bool {
	return i == Search
}

func (i ItemType) IsBinary() bool {
	// XXX: HTML from floodgap.com seems to come back as Binary rather than dotproto:
	// gopher://gopher.floodgap.com/hURL:http://gopher.floodgap.com/overbite/
	return i == DOSArchive ||
		i == Binary ||
		i == HTML ||
		i == GIF
}
