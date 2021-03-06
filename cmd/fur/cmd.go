package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/arg"
	"github.com/shabbyrobe/cmdy/flags"
	"github.com/shabbyrobe/fur/internal/furball"
	"github.com/shabbyrobe/furlib/gopher"
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

type urlVar gopher.URL

func (uv urlVar) URL() gopher.URL {
	return gopher.URL(uv)
}

func (uv urlVar) String() string {
	return gopher.URL(uv).String()
}

func (uv *urlVar) Set(s string) error {
	if s == "veronica2" || s == "search" {
		s = "gopher://gopher.floodgap.com/7/v2/vs"
	}
	u, err := gopher.ParseURL(s)
	if err != nil {
		return err
	}
	*uv = urlVar(u)
	return nil
}

type command struct {
	timeout time.Duration
	url     urlVar

	raw bool // Raw mode
	txt bool // Raw text mode

	search      string
	json        bool
	meta        bool
	allMeta     bool
	outFile     string
	outAutoFile bool
	tlsInsist   bool
	tlsDisabled bool
	format      string
	w3m         string
	htmlMode    string
	upscale     bool
	insecure    bool
	lcut        int
	include     flags.StringList
	exclude     flags.StringList
	maxEmpty    int
	cols        int
	ballFile    string
	ball        *furball.Ball
	tor         bool
	spam        int
	spamWorkers int
	stats       bool
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
				Command: `-tx=i -j gopher.floodgap.com/1/world`,
			},
		},
	}
}

func (cmd *command) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	flags.BoolVar(&cmd.raw, "raw", false, ``+
		`Raw mode; bypass all fancy rendering and print the raw bytes off the wire (will include '.\r\n' termination lines if present). Exclusive with -txt.`)
	flags.BoolVar(&cmd.txt, "txt", false, ``+
		`Raw text mode; bypass all fancy rendering, but decode as text (dot-escaped). Exclusive with -raw.`)
	flags.StringVar(&cmd.ballFile, "ball", "", ``+
		`Append request to this furball (like HAR, but crappier)`)

	// Useful for servers that misuse the '1' item type and just prepend 'i' to every line
	// of a random file regardless of what it contains:
	flags.IntVar(&cmd.lcut, "lcut", 0, "In raw mode, cut this many chars off the left.")

	flags.IntVar(&cmd.cols, "cols", 0, "Wrap columns, 0 to detect")
	flags.IntVar(&cmd.maxEmpty, "maxempty", 2, "Maximum number of empty 'i' lines to print in a row (0 = unlimited)")
	flags.BoolVar(&cmd.upscale, "upscale", true, "Upscale images")
	flags.BoolVar(&cmd.insecure, "noverify", false, "Insecure TLS - skip hostname verification")
	flags.BoolVar(&cmd.json, "j", false, "Render as JSON; will show base64 for binary, string for text and jsonl/ndjson for directories")
	flags.BoolVar(&cmd.meta, "meta", false, "Request GopherIIbis metadata for this file")
	flags.BoolVar(&cmd.tor, "tor", false, "Connect via TOR (VERY slow)")
	flags.BoolVar(&cmd.allMeta, "allmeta", false, "Request GopherIIbis metadata for the entire directory")
	flags.BoolVar(&cmd.outAutoFile, "O", false, "Output to file, infer name from selector")
	flags.BoolVar(&cmd.stats, "stats", true, "Print stats to stderr after render")
	flags.BoolVar(&cmd.tlsInsist, "tls", false, "Insist on TLS")
	flags.BoolVar(&cmd.tlsDisabled, "notls", false, "Do not attempt to automatically connect using TLS")
	flags.DurationVar(&cmd.timeout, "t", 20*time.Second, "Timeout")
	flags.StringVar(&cmd.outFile, "o", "", "Output file")
	flags.StringVar(&cmd.search, "search", "", "Search (overrides URL)")
	flags.StringVar(&cmd.format, "format", "", "GopherIIbis 'format' (content-typeish) request. Not valid with -search")
	flags.StringVar(&cmd.w3m, "w3m", "", "Path to w3m for HTML rendering (detects)")
	flags.StringVar(&cmd.htmlMode, "html", "godown", "HTML mode (godown, w3m)")
	flags.Var(&cmd.include, "ti", "Include these item types. Pass as a string, no spaces or commas. Can pass multiple times. -x=12 is the same as -x=1 -x=2")
	flags.Var(&cmd.exclude, "tx", "Exclude these item types. Takes precedence over -i. See -i for details.")

	flags.IntVar(&cmd.spam, "spam", 0, ""+
		"Spam the URL with this many requests, print stats. Similar to 'ab'. Don't use on servers that aren't yours to spam.")
	flags.IntVar(&cmd.spamWorkers, "workers", 10, ""+
		"Number of workers to use when spamming.")

	args.Var(&cmd.url, "url", "Gopher url (e.g. 'gopher://gopher.floodgap.com'). Scheme is optional. Can also use the alias 'search' to search against Veronica2.")
	args.StringOptional(&cmd.search, "search", "", "Search (overrides search portion of URL)")
}

func (cmd *command) URL() (gopher.URL, error) {
	u := cmd.url.URL()
	if cmd.search != "" {
		u.Search = cmd.search
	}
	if cmd.meta {
		u = u.AsMetaItem()
	} else if cmd.allMeta {
		u = u.AsMetaDir()
	}
	if u.ItemType.IsSearch() && u.Search == "" {
		return u, fmt.Errorf("this item type requires a search term; use --search or <search>, or add a search term to the URL")
	}
	return u, nil
}

func (cmd *command) Client(ctx context.Context) (*gopher.Client, DoneFunc, error) {
	done := nilDone
	client := &gopher.Client{
		Timeout: cmd.timeout,
		TLSMode: gopher.TLSWithInsecure,
	}
	if cmd.tlsInsist {
		client.TLSMode = gopher.TLSInsist
	} else if cmd.tlsDisabled {
		client.TLSMode = gopher.TLSDisabled
	}
	if cmd.ball != nil {
		client.Recorder = cmd.ball // 'nil interface' hazard
	}
	if cmd.insecure {
		client.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	if cmd.tor {
		t, err := tor.Start(nil, nil)
		if err != nil {
			return nil, done, err
		}
		dialer, err := t.Dialer(ctx, nil)
		if err != nil {
			return nil, done, err
		}
		client.DialContext = dialer.DialContext

		done = func() { t.Close() }
	}

	return client, done, nil
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

func (cmd *command) Run(ctx cmdy.Context) (err error) {
	if cmd.spam <= 0 && cmd.ballFile != "" {
		cmd.ball, err = furball.LoadBallFile(cmd.ballFile)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("fur: could not load ball %q: %w", cmd.ball, err)
		}
		if cmd.ball == nil {
			cmd.ball = &furball.Ball{}
		}

		n := len(cmd.ball.Entries)
		defer func() {
			if len(cmd.ball.Entries) != n {
				if serr := furball.SaveBallFile(cmd.ball, cmd.ballFile); serr != nil && err == nil {
					err = serr
				}
			}
		}()
	}

	if cmd.spam > 0 {
		return cmd.runSpam(ctx)
	} else if cmd.raw {
		return cmd.runRaw(ctx, true)
	} else if cmd.txt {
		return cmd.runRaw(ctx, false)
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
		return nil, false, fmt.Errorf("unknown response type %s", rs.Info().URL().ItemType)
	}

	return rnd, allowDefaultStdout, nil
}

func (cmd *command) selectTextRenderer(rs gopher.Response) (rnd renderer, allowDefaultStdout bool, err error) {
	cols, _ := cmd.termSize()

	url := rs.Info().URL()

	allowDefaultStdout = true
	switch rs.(type) {
	case *gopher.DirResponse:
		rnd = &dirRenderer{maxEmpty: cmd.maxEmpty, items: cmd.itemSet(), cols: cols}

	case *gopher.TextResponse:
		switch url.ItemType {
		case gopher.HTML:
			rnd = &htmlRenderer{mode: cmd.htmlMode, w3m: cmd.w3m, cols: cols}
		default:
			rnd = &rawRenderer{}
		}

	case *gopher.BinaryResponse:
		switch url.ItemType {
		case gopher.GIF, gopher.Image:
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
		return nil, false, fmt.Errorf("unknown response type %s", url.ItemType)
	}

	return rnd, allowDefaultStdout, nil
}

func (cmd *command) request(u gopher.URL) (rq *gopher.Request, err error) {
	if cmd.format != "" {
		if cmd.meta || cmd.allMeta {
			return nil, fmt.Errorf("gopher: meta and format requests are mutually exclusive")
		}
		rq, err = gopher.NewFormatRequest(u, cmd.format, nil)
		if err != nil {
			return nil, err
		}
	} else {
		rq = gopher.NewRequest(u, nil)
	}
	return rq, nil
}

func (cmd *command) runClient(ctx cmdy.Context) (rerr error) {
	u, err := cmd.URL()
	if err != nil {
		return err
	}

	rq, err := cmd.request(u)
	if err != nil {
		return err
	}

	client, done, err := cmd.Client(ctx)
	defer done()
	if err != nil {
		return err
	}

	var gopherErr *gopher.Error

	start := time.Now()

	rs, err := client.Fetch(ctx, rq)
	if errors.As(err, &gopherErr) {
		return cmdy.ErrWithCode(exitCode(gopherErr.Status, 2), err)
	} else if err != nil {
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

	if err := rnd.Render(out, rs); err != nil {
		return nil
	}

	taken := time.Since(start)
	if cmd.stats {
		fmt.Fprintf(ctx.Stderr(), "  -- took %s, tls: %v --  \n", taken, rs.Info().TLS != nil)
	}
	return nil
}

func (cmd *command) runRaw(ctx cmdy.Context, bin bool) (rerr error) {
	client, done, err := cmd.Client(ctx)
	defer done()
	if err != nil {
		return err
	}

	u, err := cmd.URL()
	if err != nil {
		return err
	}

	rq, err := cmd.request(u)
	if err != nil {
		return err
	}

	rs, err := client.Raw(ctx, rq)
	if err != nil {
		return err
	}
	defer rs.Close()
	var rdr io.Reader = rs.Reader()
	if !bin {
		rdr = gopher.NewTextReader(rdr)
	}

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
		if _, err := io.Copy(out, rdr); err != nil {
			return err
		}

	} else {
		return copyWithLcut(out, rdr, cmd.lcut)
	}

	return nil
}

func (cmd *command) runSpam(ctx cmdy.Context) (rerr error) {
	// XXX: this is a cheap and nasty rough cut of a spammer; it needs a lot of extra
	// work.

	if cmd.spamWorkers <= 0 {
		return fmt.Errorf("spam workers must be > 0")
	}

	u, err := cmd.URL()
	if err != nil {
		return err
	}

	client, done, err := cmd.Client(ctx)
	defer done()
	if err != nil {
		return err
	}

	if client.TLSClientConfig == nil {
		client.TLSClientConfig = &tls.Config{}
	}
	client.TLSClientConfig.ClientSessionCache = tls.NewLRUClientSessionCache(1000)

	stderr := ctx.Stderr()

	var (
		failedRequest int64
		failedGeneral int64
		failedRead    int64
		totalUsec     int64 // fixme: usec might even be too coarse
	)

	// FIXME: grab should handle stats and report progress
	grab := func() {
		var gopherErr *gopher.Error
		start := time.Now()
		rq, err := cmd.request(u)
		if err != nil {
			panic(err)
		}

		rs, err := client.Fetch(ctx, rq)
		if errors.As(err, &gopherErr) {
			atomic.AddInt64(&failedRequest, 1)

		} else if err != nil {
			atomic.AddInt64(&failedGeneral, 1)

		} else if rs != nil {
			if _, err := io.Copy(ioutil.Discard, rs.Reader()); err != nil {
				atomic.AddInt64(&failedRead, 1)
			}
			rs.Close()
		}
		took := time.Since(start)
		atomic.AddInt64(&totalUsec, int64(took/time.Microsecond))
	}

	fmt.Fprintf(stderr, "spamming %d requests with %d workers\n", cmd.spam, cmd.spamWorkers)
	fmt.Fprintf(stderr, "%q\n", u)

	var left = int64(cmd.spam)
	var workerDone = make(chan struct{}, cmd.spamWorkers)
	var progress = make(chan int64)

	for i := 0; i < cmd.spamWorkers; i++ {
		go func() {
			defer func() {
				workerDone <- struct{}{}
			}()
			for {
				n := atomic.AddInt64(&left, -1)
				if n <= 0 {
					break
				}
				grab()
			}
		}()
	}

	printStats := func(rqLeft int64) {
		n := int64(cmd.spam) - rqLeft
		usec := atomic.LoadInt64(&totalUsec)
		if usec == 0 {
			usec = 1
		}
		avg := float64(usec) / float64(n) / 1000
		tps := float64(n) / (float64(usec) / float64(cmd.spamWorkers) / 1000000)
		fmt.Printf("%d avgms:%.2f tps:%.0f\n", n, avg, tps)
	}

	workersLeft := cmd.spamWorkers
	for workersLeft > 0 {
		select {
		case pleft := <-progress:
			printStats(pleft)

		case <-workerDone:
			workersLeft--

		case <-ctx.Done():
			atomic.StoreInt64(&left, 0)
		}
	}

	for workersLeft > 0 {
		<-workerDone
		workersLeft--
	}

	printStats(left)
	fmt.Printf("failreq:%d failgen:%d failrd:%d\n", failedRequest, failedGeneral, failedRead)

	return nil
}

type DoneFunc func()

var nilDone DoneFunc = func() {}
