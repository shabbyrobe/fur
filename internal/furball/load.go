package furball

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

func LoadBallFile(path string) (*Ball, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("furball: load file %q failed: %w", path, err)
	}
	defer f.Close()

	var b Ball
	dec := json.NewDecoder(f)
	if err := dec.Decode(&b); err != nil {
		return nil, err
	}
	return &b, err
}

func SaveBallFile(ball *Ball, path string) (rerr error) {
	var b [16]byte
	rand.Read(b[:])

	tmpPath := path + "." + hex.EncodeToString(b[:])
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("furball: load file %q failed: %w", path, err)
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(ball); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	f = nil

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	return nil
}
