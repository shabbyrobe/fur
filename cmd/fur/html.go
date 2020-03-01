package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"github.com/mattn/godown"
	"github.com/shabbyrobe/fur/internal/mdhtml"
	"github.com/shabbyrobe/furlib/gopher"
)

type htmlRenderer struct {
	w3m  string
	mode string
	cols int
}

func (d *htmlRenderer) Render(out io.Writer, rs gopher.Response) error {
	rrs := rs.(io.Reader)

	switch d.mode {
	case "w3m":
		w3m := d.w3m
		if d.w3m == "" {
			wp, err := exec.LookPath("w3m")
			if err != nil {
				_, err := io.Copy(out, rrs)
				return err
			}
			w3m = wp
		}
		cmd := exec.Command(w3m, "-T", "text/html", "-dump")
		cmd.Stdin = rrs
		cmd.Stdout = out
		return cmd.Run()

	case "godown":
		var buf bytes.Buffer
		if err := godown.Convert(&buf, rrs, &godown.Option{Script: false}); err != nil {
			return err
		}

		result := mdhtml.Render(buf.String(), d.cols, 0)
		_, err := out.Write(result)
		return err

	default:
		return fmt.Errorf("unknown html mode %q", d.mode)
	}
}
