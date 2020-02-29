package gopher

import (
	"context"
	"time"
)

type Caps interface {
	Version() int
	ExpiresAfter() time.Duration // Return -1 if the Caps don't expire

	Supports(feature Feature) FeatureStatus
	PathConfig() (*PathConfig, error)
	ServerInfo() (*ServerInfo, error)
	Software() (name, version string)

	// TLSPort for the server; returns 0 if not present or configured.
	TLSPort() int

	// Default text encoding for content types 0 and 1.
	// If this returns an empty string, UTF-8 is presumed.
	DefaultEncoding() string
}

type CapsSource interface {
	LoadCaps(ctx context.Context, host, port string) (Caps, error)
}

type CapsUpdater interface {
	UpdateFeature(ctx context.Context, host, port string, feature Feature)
}

var DefaultCaps Caps = defaultCaps{}

type defaultCaps struct{}

var _ Caps = defaultCaps{}

func (defaultCaps) Version() int                           { return 1 }
func (defaultCaps) ExpiresAfter() time.Duration            { return -1 }
func (defaultCaps) Supports(feature Feature) FeatureStatus { return FeatureUnsupported }
func (defaultCaps) ServerInfo() (*ServerInfo, error)       { return nil, nil }
func (defaultCaps) Software() (name, version string)       { return "", "" }
func (defaultCaps) DefaultEncoding() string                { return "UTF-8" }
func (defaultCaps) TLSPort() int                           { return 0 }

func (defaultCaps) PathConfig() (*PathConfig, error) {
	up := UnixPathConfig
	return &up, nil
}
