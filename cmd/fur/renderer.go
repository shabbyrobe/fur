package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/bbrks/wrap"
	"github.com/shabbyrobe/fur/internal/gopher"
)

var defaultIcons = [256]rune{
	gopher.Search:        '🔍',
	gopher.Dir:           '📂',
	gopher.Text:          '📄',
	gopher.Sound:         '🔈',
	gopher.CSOServer:     '📞',
	gopher.Image:         '📷',
	gopher.GIF:           '📷',
	gopher.Binary:        '💾',
	gopher.BinaryArchive: '💾',
	gopher.BinHex:        '💾',
	gopher.UUEncoded:     '💾',
	gopher.Telnet:        '📺',
	gopher.TN3270:        '📺',
}

var itemColors = [256]string{}

func init() {
	add, start, limit := 61, 16, 216 // add must be odd
	for i, v := 0, 0; i < 256; i++ {
		itemColors[i] = fmt.Sprintf("\033[38;5;%dm", start+v)
		v = (v + add) % limit
	}
	itemColors[gopher.Dir] = "\033[38;5;118m"
	itemColors[gopher.ItemError] = "\033[38;5;196m"
	itemColors[gopher.HTML] = "\033[38;5;226m"
	itemColors[gopher.Info] = "\033[38;5;241m"
}

type renderer interface {
	Render(out io.Writer, rs gopher.Response) error
}

type rawRenderer struct{}

func (d *rawRenderer) Render(out io.Writer, rs gopher.Response) error {
	rrs := rs.(io.Reader)
	_, err := io.Copy(out, rrs)
	return err
}

type dirRenderer struct {
	items    [256]bool
	icons    *[256]rune
	cols     int
	maxEmpty int
}

var _ renderer = &dirRenderer{}

func (d *dirRenderer) Render(out io.Writer, rs gopher.Response) error {
	drs := rs.(*gopher.DirResponse)

	icons := d.icons
	if icons == nil {
		icons = &defaultIcons
	}

	var dirent gopher.Dirent

	const indent = "   "
	var wrp = wrap.NewWrapper()
	wrp.OutputLinePrefix = indent

	i := 0
	emptyLeft := d.maxEmpty

	var lset byteSet

	for drs.Next(&dirent) {
		if !d.items[dirent.ItemType] {
			continue
		}

		if d.maxEmpty > 0 {
			if dirent.ItemType == gopher.Info && strings.TrimSpace(dirent.Display) == "" {
				emptyLeft--
			} else {
				emptyLeft = d.maxEmpty
			}
			if emptyLeft <= 0 {
				continue
			}
		}

		dwrap := dirent.Display
		dwrap = wrp.Wrap(dwrap, d.cols)
		dwrap = strings.TrimRight(dwrap, "\r\n\t ")
		if strings.HasPrefix(dwrap, indent) {
			// bbrks/wrap always puts a leading indent
			dwrap = dwrap[len(indent):]
		}

		switch dirent.ItemType {
		case gopher.Info:
			// XXX: We can get a lot fancier than this if we buffer a few dirents; we can
			// scale the threshold down based on how many lines before or after contain
			// ASCII. We can also try to detect HRs and color them separately.
			if asciiConfidence(dirent.Display, &lset) > 0.5 {
				fmt.Fprintf(out, "%s\033[38;5;96m%s\033[m\n", indent, dwrap)
			} else {
				fmt.Fprintf(out, "%s\033[38;5;250m%s\033[m\n", indent, dwrap)
			}

		default:
			var c rune = icons[dirent.ItemType]
			if c == 0 {
				fmt.Fprintf(out, "%s%c\033[m) ", itemColors[dirent.ItemType], dirent.ItemType)
			} else {
				fmt.Fprintf(out, "%c ", c)
			}

			urlStr, ok := dirent.URL.WWW()
			if !ok {
				urlStr = dirent.URL.String()
			}

			lwspn := lwspCount(dirent.Display)
			lwsp := dirent.Display[:lwspn]

			fmt.Fprintf(out, "%s\n", dwrap)
			fmt.Fprintf(out, "%s%s  \033[38;5;45m└─ %s\033[m\n", lwsp, indent, urlStr)
		}

		i++
	}

	return nil
}

func lwspCount(s string) int {
	sl := len(s)
	i := 0
	for ; i < sl; i++ {
		if s[i] != ' ' && s[i] != '\t' {
			break
		}
	}
	return i
}

var (
	symbolSet = newByteSet("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~")
	spaceSet  = newByteSet(" \r\t\n")
)

func asciiConfidence(in string, lset *byteSet) float64 {
	lset.SetString(in)
	lset.AndNot(spaceSet)
	inCnt := lset.Count()

	lset.And(symbolSet)

	var asciiConfidence float64
	if inCnt > 0 {
		asciiConfidence = float64(lset.Count()) / float64(inCnt)
	}
	return asciiConfidence
}
