package gopher

import "strings"

type MetaType byte

const (
	MetaNone MetaType = 0
	MetaItem MetaType = '!'
	MetaDir  MetaType = '&'
)

func metaRecordSelector(meta MetaType, records ...string) string {
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
