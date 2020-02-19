//+build ignore

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/shabbyrobe/fur/internal/uuencode"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var name string

	flag.StringVar(&name, "name", "-", "encode name")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return fmt.Errorf("usage (dec|enc)")
	}

	switch args[0] {
	case "enc":
		uw := uuencode.NewWriter(os.Stdout, name, 0664)
		if _, err := io.Copy(uw, os.Stdin); err != nil {
			return err
		}
		if err := uw.Flush(); err != nil {
			return err
		}

	case "dec":
		ur := uuencode.NewReader(os.Stdin, nil)
		_, err := io.Copy(os.Stdout, ur)
		f, _ := ur.File()
		m, _ := ur.Mode()
		fmt.Fprintf(os.Stderr, "file: %s, mode: %03o\n", f, m)
		return err
	}

	return nil
}
