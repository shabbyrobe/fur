package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/arg"
	"github.com/shabbyrobe/cmdy/flags"
	"github.com/shabbyrobe/fur/internal/gopher"
)

const commandUsage = `
"Fur" is a command-line Gopher client that probably doesn't work very well.

Gopher URLs are in the following format:
    gopher://<host>[:<port>]/1selector
    gopher://<host>[:<port>]/1selector%09search
    gopher://<host>[:<port>]/1selector%09search%09plus

The port is optional, and defaults to ':70'. The 'search' portion can also be provided
via the '--search' flag or the '<search>' argument.
`

type command struct {
	timeout     time.Duration
	url         string
	raw         bool
	search      string
	json        bool
	outFile     string
	outAutoFile bool
	w3m         string
	htmlMode    string
	upscale     bool
	lcut        int
	include     flags.StringList
	exclude     flags.StringList
	maxEmpty    int
	cols        int
}

func (cmd *command) Help() cmdy.Help {
	return cmdy.Help{
		Synopsis: "Fur - CLI Gopher Client",
		Usage:    commandUsage,
		Examples: cmdy.Examples{
			cmdy.Example{
				Desc:    "Visit Gopherpedia (Wikipedia)",
				Command: "gopher://gopherpedia.com",
			},
			cmdy.Example{
				Desc:    "Visit Floodgap",
				Command: "gopher://gopher.floodgap.com/",
			},
			cmdy.Example{
				Desc:    "Directory of gopher servers",
				Command: "gopher://gopher.floodgap.com/1/world",
			},
			cmdy.Example{
				Desc:    "Visit SDF Public Access UNIX System",
				Command: "sdf.org",
			},
			cmdy.Example{
				Desc:    "Hacker news",
				Command: "hngopher.com",
			},
			cmdy.Example{
				Desc:    "Reddit",
				Command: "gopherddit.com",
			},
			cmdy.Example{
				Desc:    "Search gopher (multiple terms are interpreted as an 'AND' query)",
				Command: `search "term1 term2"`,
			},

			cmdy.Example{
				Desc:    "Dump directory as json, excluding 'i' types",
				Command: `-x=i -j gopher.floodgap.com/1/world`,
			},
		},
	}
}

func (cmd *command) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	flags.BoolVar(&cmd.raw, "raw", false, `Raw mode; bypass all fancy rendering and print the raw bytes (will include '.\r\n' termination lines if present)`)

	//  Useful for servers that misuse the '1' item type and just prepend 'i' to every line of a random file regardless of what it contains
	flags.IntVar(&cmd.lcut, "lcut", 0, "In raw mode, cut this many chars off the left.")

	flags.IntVar(&cmd.cols, "cols", 0, "Wrap columns, 0 to detect")
	flags.IntVar(&cmd.maxEmpty, "maxempty", 2, "Maximum number of empty 'i' lines to print in a row (0 = unlimited)")
	flags.BoolVar(&cmd.upscale, "upscale", true, "Upscale images")
	flags.BoolVar(&cmd.json, "j", false, "Render as JSON; will show base64 for binary, string for text and jsonl/ndjson for directories")
	flags.BoolVar(&cmd.outAutoFile, "O", false, "Output to file, infer name from selector")
	flags.DurationVar(&cmd.timeout, "t", 15*time.Second, "Timeout")
	flags.StringVar(&cmd.outFile, "o", "", "Output file")
	flags.StringVar(&cmd.search, "search", "", "Search (overrides URL)")
	flags.StringVar(&cmd.w3m, "w3m", "", "Path to w3m for HTML rendering (detects)")
	flags.StringVar(&cmd.htmlMode, "html", "godown", "HTML mode (godown, w3m)")
	flags.Var(&cmd.include, "i", "Include these item types. Pass as a string, no spaces or commas. Can pass multiple times. -x=12 is the same as -x=1 -x=2")
	flags.Var(&cmd.exclude, "x", "Exclude these item types. Takes precedence over -i. See -i for details.")
	args.String(&cmd.url, "url", "Gopher url (e.g. 'gopher://gopher.floodgap.com'). Scheme is optional. Can also use the alias 'search' to search against Veronica2.")
	args.StringOptional(&cmd.search, "search", "", "Search (overrides search portion of URL)")
}

func (cmd *command) URL() (gopher.URL, error) {
	ustr := cmd.url
	if ustr == "veronica2" || ustr == "search" {
		ustr = "gopher://gopher.floodgap.com/7/v2/vs"
	}
	u, err := gopher.ParseURL(ustr)
	if err != nil {
		return u, err
	}
	if cmd.search != "" {
		u.Search = cmd.search
	}
	return u, nil
}

func (cmd *command) Client() *gopher.Client {
	return &gopher.Client{
		Timeout: cmd.timeout,
	}
}

func (cmd *command) outFileName(u gopher.URL) string {
	if cmd.outFile != "" {
		return cmd.outFile

	} else if cmd.outAutoFile {
		base := path.Base(u.Selector)
		if base == "" || base == "/" || base == "." {
			return ""
		}

		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		curBase := base
		for i := 1; ; i++ {
			full := filepath.Join(wd, curBase)
			if _, err := os.Stat(full); errors.Is(err, os.ErrNotExist) {
				return curBase
			}
			curBase = fmt.Sprintf("%s.%d", base, i)
		}

	} else {
		return ""
	}
}

func (cmd *command) Run(ctx cmdy.Context) error {
	if cmd.raw {
		return cmd.runRaw(ctx)
	} else {
		return cmd.runClient(ctx)
	}
	return nil
}

func (cmd *command) itemSet() [256]bool {
	set := [256]bool{}

	if len(cmd.include) > 0 {
		for _, inc := range cmd.include {
			for i := 0; i < len(inc); i++ {
				set[inc[i]] = true
			}
		}
	} else {
		for i := 0; i < 256; i++ {
			set[i] = true
		}
	}

	for _, exc := range cmd.exclude {
		for i := 0; i < len(exc); i++ {
			set[exc[i]] = false
		}
	}

	return set
}

func (cmd *command) termSize() (cols, rows int) {
	cols, rows = termSize()
	if cmd.cols != 0 {
		cols = cmd.cols
	}
	return cols, rows
}

func (cmd *command) selectRenderer(rs gopher.Response) (rnd renderer, allowDefaultStdout bool, err error) {
	if cmd.json {
		return cmd.selectJSONRenderer(rs)
	}
	return cmd.selectTextRenderer(rs)
}

func (cmd *command) selectJSONRenderer(rs gopher.Response) (rnd renderer, allowDefaultStdout bool, err error) {
	allowDefaultStdout = true

	switch rs := rs.(type) {
	case *gopher.DirResponse:
		rnd = &jsonDirRenderer{items: cmd.itemSet()}
	case *gopher.TextResponse:
		rnd = &jsonTextRenderer{}
	case *gopher.BinaryResponse:
		rnd = &jsonBinaryRenderer{}
	case *gopher.UUEncodedResponse:
		rnd = &jsonBinaryRenderer{}
	default:
		return nil, false, fmt.Errorf("unknown response type %s", rs.URL().ItemType)
	}

	return rnd, allowDefaultStdout, nil
}

func (cmd *command) selectTextRenderer(rs gopher.Response) (rnd renderer, allowDefaultStdout bool, err error) {
	cols, _ := cmd.termSize()

	allowDefaultStdout = true
	switch rs := rs.(type) {
	case *gopher.DirResponse:
		rnd = &dirRenderer{maxEmpty: cmd.maxEmpty, items: cmd.itemSet(), cols: cols}

	case *gopher.TextResponse:
		switch rs.URL().ItemType {
		case gopher.HTML:
			rnd = &htmlRenderer{mode: cmd.htmlMode, w3m: cmd.w3m, cols: cols}
		default:
			rnd = &rawRenderer{}
		}

	case *gopher.BinaryResponse:
		switch rs.URL().ItemType {
		case gopher.GIF:
			rnd = &imageRenderer{upscale: cmd.upscale}
		case gopher.HTML:
			rnd = &htmlRenderer{mode: cmd.htmlMode, w3m: cmd.w3m, cols: cols}
		default:
			allowDefaultStdout = false
			rnd = &rawRenderer{}
		}

	case *gopher.UUEncodedResponse:
		allowDefaultStdout = false
		rnd = &rawRenderer{}

	default:
		return nil, false, fmt.Errorf("unknown response type %s", rs.URL().ItemType)
	}

	return rnd, allowDefaultStdout, nil
}

func (cmd *command) runClient(ctx cmdy.Context) (rerr error) {
	u, err := cmd.URL()
	if err != nil {
		return err
	}

	if u.ItemType.IsSearch() && u.Search == "" {
		return fmt.Errorf("this item type requires a search term; use --search or <search>, or add a search term to the URL")
	}

	client := cmd.Client()

	rs, err := client.Fetch(ctx, u)
	if err != nil {
		return err
	}
	defer DeferClose(&rerr, rs)

	rnd, allowDefaultStdout, err := cmd.selectRenderer(rs)
	if err != nil {
		return err
	}

	outFile := cmd.outFileName(u)
	out, isFile, err := stdoutOrFileWriter(ctx.Stdout(), outFile, allowDefaultStdout)
	if err != nil {
		return err
	}
	defer DeferClose(&rerr, out)

	if isFile {
		fmt.Fprintf(ctx.Stderr(), "writing to %q\n", outFile)
	}

	return rnd.Render(out, rs)
}

func (cmd *command) runRaw(ctx cmdy.Context) (rerr error) {
	client := cmd.Client()

	u, err := cmd.URL()
	if err != nil {
		return err
	}

	rs, err := client.Binary(ctx, u)
	if err != nil {
		return err
	}
	defer rs.Close()

	outFile := cmd.outFileName(u)
	out, isFile, err := stdoutOrFileWriter(ctx.Stdout(), outFile, true)
	if err != nil {
		return err
	}
	defer DeferClose(&rerr, out)

	if isFile {
		fmt.Fprintf(ctx.Stderr(), "writing to %q\n", outFile)
	}

	if cmd.lcut == 0 {
		if _, err := io.Copy(out, rs); err != nil {
			return err
		}

	} else {
		return copyWithLcut(out, rs, cmd.lcut)
	}

	return nil
}

func copyWithLcut(out io.Writer, rs io.Reader, lcut int) error {
	var scratch = make([]byte, 8192)
	var buf = make([]byte, 0, 8192)
	var readDone = false
	var pos = 0

	for !readDone {
		var idx = -1
		for idx < 0 && !readDone {
			buf = buf[:copy(buf, buf[pos:])]
			pos = 0

			n, err := rs.Read(scratch)
			if err != nil && err != io.EOF {
				return err
			}
			buf = append(buf, scratch[:n]...)
			if err == io.EOF {
				readDone = true
			}
			idx = bytes.IndexByte(buf, '\n')
		}

	again:
		var line []byte
		if idx >= 0 {
			end := pos + idx + 1
			line = buf[pos:end]
			pos = end
		} else if readDone {
			line = buf[pos:]
		} else {
			continue
		}

		if len(line) > 0 {
			lmax := len(line)
			if lmax > 0 && line[lmax-1] == '\r' {
				lmax--
			}
			cut := lcut
			if cut > lmax {
				cut = lmax
			}
			line = line[cut:]
			out.Write(line)
		}

		if !readDone {
			idx = bytes.IndexByte(buf[pos:], '\n')
			goto again
		}
	}

	return nil
}
