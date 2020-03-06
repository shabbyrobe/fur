package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/shabbyrobe/furlib/gopher"
)

type jsonDirRenderer struct {
	items [256]bool
}

var _ renderer = &jsonDirRenderer{}

func (jd *jsonDirRenderer) Render(out io.Writer, rs gopher.Response) error {
	rrs := rs.(*gopher.DirResponse)
	enc := json.NewEncoder(out)
	var dirent gopher.Dirent
	for rrs.Next(&dirent) {
		if !jd.items[dirent.ItemType] {
			continue
		}
		if err := enc.Encode(&dirent); err != nil {
			return err
		}
	}
	return nil
}

type jsonTextRenderer struct {
}

var _ renderer = &jsonTextRenderer{}

func (jd *jsonTextRenderer) Render(out io.Writer, rs gopher.Response) error {
	data, err := ioutil.ReadAll(rs.(io.Reader))
	if err != nil {
		return err
	}
	enc, err := json.Marshal(string(data))
	if err != nil {
		return err
	}
	out.Write(enc)
	return nil
}

type jsonBinaryRenderer struct {
}

var _ renderer = &jsonBinaryRenderer{}

func (jd *jsonBinaryRenderer) Render(out io.Writer, rs gopher.Response) error {
	rrs := rs.(io.Reader)
	io.WriteString(out, `"`)
	enc := base64.NewEncoder(base64.StdEncoding, out)
	defer enc.Close()
	if _, err := io.Copy(enc, rrs); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	io.WriteString(out, `"`)
	return nil
}
