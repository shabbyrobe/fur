uuencode/uudecode
=================

I made this to decode UUEncoded selectors (item type '6') from Gopher servers...
and then couldn't find a single UUEncoded thing to test it with.

I stopped bothering to polish it up after that. The fuzz tester found a lot of
bugs but there will be more. Also would be good to get rid of that bufio.Writer
in the encoder.

This was actually a lot trickier than I expected to get right - `Read()` can be
a bit of a beast when every single byte could be in the middle of a big set of
different states.

