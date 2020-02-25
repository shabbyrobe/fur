package gopher

type MetaType byte

const (
	MetaNone MetaType = 0
	MetaItem MetaType = '!'
	MetaDir  MetaType = '&'
)
