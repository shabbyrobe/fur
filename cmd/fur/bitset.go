package main

import (
	"fmt"
	"math/bits"
	"strings"
)

type byteSet [4]uint64

func newByteSet(s string) byteSet {
	var bs byteSet
	if s != "" {
		bs.SetString(s)
	}
	return bs
}

func (b byteSet) Dump() string {
	var bld strings.Builder
	first := true
	for i := 0; i < 256; i++ {
		if b.IsSet(byte(i)) {
			if !first {
				bld.WriteString(", ")
			}
			bld.WriteString(fmt.Sprintf("0x%02x", i))
			first = false
		}
	}
	return bld.String()
}

func (b *byteSet) SetString(s string) {
	*b = [4]uint64{}
	sl := len(s)
	for i := 0; i < sl; i++ {
		b[s[i]>>6] |= 1 << (s[i] & 63)
	}
}

func (b *byteSet) Clear() {
	*b = [4]uint64{}
}

func (b *byteSet) Set(v byte) {
	b[v>>6] |= 1 << (v & 63)
}

func (b *byteSet) Unset(v byte) {
	b[v>>6] &= ^(1 << (v & 63))
}

func (b byteSet) IsSet(v byte) bool {
	return b[v>>6]&(1<<(v&63)) != 0
}

func (b *byteSet) And(v byteSet) {
	b[0], b[1], b[2], b[3] = b[0]&v[0], b[1]&v[1], b[2]&v[2], b[3]&v[3]
}

func (b *byteSet) AndNot(v byteSet) {
	b[0], b[1], b[2], b[3] = b[0]&^v[0], b[1]&^v[1], b[2]&^v[2], b[3]&^v[3]
}

func (b *byteSet) Or(v byteSet) {
	b[0], b[1], b[2], b[3] = b[0]|v[0], b[1]|v[1], b[2]|v[2], b[3]|v[3]
}

func (b *byteSet) Invert() {
	b[0], b[1], b[2], b[3] = ^b[0], ^b[1], ^b[2], ^b[3]
}

func (b byteSet) SubsetOf(set byteSet) bool {
	return set[0]&b[0] == b[0] &&
		set[1]&b[1] == b[1] &&
		set[2]&b[2] == b[2] &&
		set[3]&b[3] == b[3]
}

func (b byteSet) Count() (count int) {
	return bits.OnesCount64(b[0]) +
		bits.OnesCount64(b[1]) +
		bits.OnesCount64(b[2]) +
		bits.OnesCount64(b[3])
}
