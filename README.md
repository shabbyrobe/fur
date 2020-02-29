Fur: command-line Gopher client
===============================

This is my crappy command line Gopher client.

This project exists only as a scratchpad for me to experiment with the Gopher
protocol, which I am interested in/fascinated by for absolutely no good reason
whatsoever. There are better libraries and tools around, with better levels of
completeness. Probably better to use those.

I shoved some nice things in here because it was easy:

- HTML rendering using (a fork of) https://github.com/MichaelMure/go-term-markdown/
- Image rendering for `I` and `g` types using https://github.com/shabbyrobe/termimg/
- Automatic unpacking of UUEncoded (`6`) types, which I added before I noticed that
  I can't find anything that uses `6` anywhere.
- Error detection

## Expectation Management

Feel free to use this or take bits from it as you see fit (MIT license == go
nuts). I won't maintain this to any kind of standard though. This is a
scratchpad and a bit of fun for me, not a product. Issues may be responded to
whenever I happen to get around to them, but PRs are unlikely to be accepted.

## Install

Source only:

    go install ./cmd/fur

## Using

Easy!

    $ fur gopher.floodgap.com
    $ fur gopher://gopher.floodgap.com
    $ fur hngopher.com
    $ fur -tx=i search "hacker news"

Then if you see a link, just copy and paste it in a subsequent invocation to `fur`.

To get the raw output, use the `--raw` flag.

HTML item types (`h`) work best if you have `w3m` installed.


## Links

### Gopher sites:

- [GopherPedia](gopher://gopherpedia.com/)
- [Floodgap](gopher://gopher.floodgap.com/)
- [SDF Public Access UNIX System](gopher://sdf.org/)
- [Large list of known gopher servers](gopher://gopher.floodgap.com/1/world)
- [Search Gopher with Veronica-2](gopher://gopher.floodgap.com/7/v2/vs)
- [Hacker News](gopher://hngopher.com/)
- [Metafilter](gopher://gopher.metafilter.com)


### Gopher history/general:

- gopher://gopher.viste.fr/1/gopher-faq
- https://tedium.co/2017/06/22/modern-day-gopher-history/
- https://ils.unc.edu/callee/gopherpaper.htm
- https://arstechnica.com/tech-policy/2009/11/the-web-may-have-won-but-gopher-tunnels-on/
- http://gopher.floodgap.com/overbite/relevance.html
- http://muffinlabs.com/2013/06/12/goper2000---a-modern-gopher-server/
- https://news.ycombinator.com/item?id=12269784
- https://www.ics.uci.edu/~rohit/IEEE-L7-http-gopher.html


### Gopher articles of interest:

- https://jfm.carcosa.net/blog/computing/hugo-gopher/


### Server software:

- https://github.com/gophernicus/gophernicus
- https://github.com/muffinista/gopher2000/
- https://github.com/knusbaum/cl-gopher
- http://motsognir.sourceforge.net/
- https://github.com/spc476/port70
- https://github.com/jgoerzen/pygopherd
- https://github.com/asterIRC/uGopherServer
- https://github.com/michael-lazar/flask-gopher
- http://gofish.sourceforge.net/
- https://github.com/ix/geomyidae
- http://r-36.net/scm/geomyidae/file/CGI.html (Basic Gopher+ meta)
- http://aftershock.sourceforge.net/ (Ancient, Java)
- http://mateusz.viste.fr/attic/grumpy/ (GPL, last updated 2011)


### Gopher protocol:

- https://tools.ietf.org/html/rfc1436
- https://sdfeu.org/w/tutorials:gopher
- gopher://gopher.floodgap.com/0/gopher/tech/gopherplus.txt
- https://tools.ietf.org/html/draft-matavka-gopher-ii-03
- https://groups.google.com/forum/#!msg/comp.sys.mac.announce/xbgsusdfETc/S2793OidrSQJ
- https://lists.debian.org/gopher-project/2018/02/msg00038.html


### TLS:

Seems that the best way to handle this is to allow clients to just talk TLS.

Might also be good to return an explicit error if a client attempts to use
STARTTLS.

- https://dataswamp.org/~solene/2019-03-07-gopher-server-tls.html
- https://alexschroeder.ch/wiki/Comments_on_2018-01-10_Encrypted_Gopher
- https://github.com/0x16h/gopher-tls/
- https://news.ycombinator.com/item?id=20171646
- https://news.ycombinator.com/item?id=16811407
- gopher://tilde.team/0/~rain1/phlog/20190608-encrypting-gopher.txt
- https://lists.debian.org/gopher-project/2018/02/msg00038.html


### Crawlers:

- gopher://gopherproject.org/1/eomyidae


### Clients:

- https://github.com/jgoerzen/gopher
- https://thelambdalab.xyz/elpher/
- https://rawtext.club/~sloum/bombadillo.html (https://tildegit.org/sloum/bombadillo)
- https://github.com/jankammerath/gophie
- https://github.com/solderpunk/VF-1 (TLS)
- https://metacpan.org/pod/release/WGDAVIS/Net-Gopher-0.43/lib/Net/Gopher.pm (Gopher+)


### Gemini:

- gopher://zaibatsu.circumlunar.space/1/~solderpunk/gemini

