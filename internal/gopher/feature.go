package gopher

type Feature int

const (
	// ASK forms from Gopher+
	FeaturePlusAsk Feature = 1

	// Server understands GopherII queries
	FeatureII Feature = 2

	// Server will respond to GopherIIbis metadata queries
	FeatureIIbis Feature = 3
)

const (
	capKeyGopherIIbis   = "SupportsGopherIIbis"
	capKeyGopherII      = "SupportsGopherII"
	capKeyGopherPlusAsk = "SupportsGopherPlusAsk"
	capKeyTLSPort       = "ServerTLSPort"
)

type FeatureStatus int

const (
	FeatureStatusUnknown FeatureStatus = iota
	FeatureSupported
	FeatureUnsupported
)

func featureStatusFromBool(v bool) FeatureStatus {
	if v {
		return FeatureSupported
	}
	return FeatureUnsupported
}
