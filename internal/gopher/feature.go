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

type FeatureStatus int

const (
	FeatureStatusUnknown FeatureStatus = iota
	FeatureSupported
	FeatureUnsupported
)
