package gopher

import "runtime"

type ServerInfo struct {
	Software     string
	Version      string
	Architecture string
	Description  string
	Geolocation  string
	AdminEmail   string
}

var defaultServerInfo = ServerInfo{
	Software:     "go/fur",
	Version:      version,
	Architecture: runtime.GOARCH,
}
