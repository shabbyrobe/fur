package uuencode

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"testing"
)

func BenchmarkDecode(b *testing.B) {
	for idx, bc := range []struct {
		datasz int
		readsz int
	}{
		{1, 2048},
		{128, 2048},
		{1024, 2048},
		{16384, 4096},
		{32768, 2048},
		{32768, 8192},
	} {
		b.Run(fmt.Sprintf("%d/datasz=%d/readsz=%d", idx, bc.datasz, bc.readsz), func(b *testing.B) {
			var bts = make([]byte, bc.datasz)
			rand.Read(bts)

			var buf bytes.Buffer
			w := NewWriter(&buf, "-", 0644)
			w.Write(bts)
			w.Flush()

			scratchNew := make([]byte, 2048)
			scratchRead := make([]byte, bc.readsz)

			in := buf.Bytes()
			b.SetBytes(int64(bc.datasz))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				rd := NewReader(bytes.NewReader(in), scratchNew)
				tot := 0
				for {
					n, err := rd.Read(scratchRead)
					tot += n
					if n == 0 && err == io.EOF {
						break
					} else if err != nil && err != io.EOF {
						b.Fatal(err)
					}
				}

				if tot != bc.datasz {
					b.Fatal(tot, "!=", bc.datasz)
				}
			}
		})
	}
}

func BenchmarkEncode(b *testing.B) {
	for idx, bc := range []struct {
		datasz int
		wrtsz  int
	}{
		{1, 2048},
		{128, 2048},
		{1024, 2048},
		{16384, 4096},
		{32768, 2048},
		{32768, 8192},
	} {
		b.Run(fmt.Sprintf("%d/datasz=%d/wrtsz=%d", idx, bc.datasz, bc.wrtsz), func(b *testing.B) {
			var bts = make([]byte, bc.datasz)
			rand.Read(bts)

			b.SetBytes(int64(bc.datasz))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				w := NewWriter(ioutil.Discard, "-", 0644)
				for p := 0; p < bc.datasz; {
					var wb []byte
					if p+bc.wrtsz > bc.datasz {
						wb = bts[p:]
					} else {
						wb = bts[p : p+bc.wrtsz]
					}
					n, _ := w.Write(wb)
					p += n
				}
				w.Flush()
			}
		})
	}
}
