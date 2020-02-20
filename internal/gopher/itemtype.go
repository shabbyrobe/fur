package gopher

import "strconv"

type ItemType byte

const (
	Text          ItemType = '0'
	Dir           ItemType = '1'
	CSOServer     ItemType = '2' // https://en.wikipedia.org/wiki/CCSO_Nameserver
	ItemError     ItemType = '3'
	BinHex        ItemType = '4' // Ancient pre OS X Mac format
	BinaryArchive ItemType = '5' // (zip; rar; 7-Zip; gzip; tar); Client must read until the TCP connection closes. Beware.
	UUEncoded     ItemType = '6'
	Search        ItemType = '7'
	Telnet        ItemType = '8' // Connect to given host at given port. The name to login as at this host is in the selector string.
	Binary        ItemType = '9' // Client must read until the TCP connection closes. Beware.

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

	// GopherII:
	Calendar = 'c'
	Doc      = 'd'
	HTML     = 'h'
	Info     = 'i'
	Page     = 'p' // e.g.  (TeX; LaTeX; PostScript; Rich Text Format)
	MBOX     = 'm' // Electronic mail repository (also known as MBOX)
	Sound    = 's'
	XML      = 'x'
	Video    = ';'
)

type Class int

const (
	UnknownClass  Class = 0
	BinaryClass   Class = 1
	ExternalClass Class = 2
	DirClass      Class = 3
	TextClass     Class = 4
	ErrorClass    Class = 5
	InfoClass     Class = 6
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

func (i ItemType) Class() Class {
	// XXX: HTML from floodgap.com seems to come back as Binary rather than dotproto:
	// gopher://gopher.floodgap.com/hURL:http://gopher.floodgap.com/overbite/
	return defaultClass[i]
}

func (i ItemType) IsBinary() bool {
	return i.Class() == BinaryClass
}

var defaultClass = [256]Class{
	Text:          TextClass,
	Dir:           DirClass,
	CSOServer:     ExternalClass,
	ItemError:     ErrorClass,
	BinHex:        TextClass,
	BinaryArchive: BinaryClass,
	UUEncoded:     TextClass,
	Search:        DirClass,
	Telnet:        ExternalClass,
	Binary:        BinaryClass,
	Duplicate:     UnknownClass,
	GIF:           BinaryClass,
	Image:         BinaryClass,
	TN3270:        ExternalClass,
	Calendar:      BinaryClass,
	Doc:           BinaryClass,
	HTML:          TextClass,
	Info:          InfoClass,
	Page:          TextClass,
	MBOX:          BinaryClass,
	Sound:         BinaryClass,
	XML:           TextClass,
	Video:         BinaryClass,
}
