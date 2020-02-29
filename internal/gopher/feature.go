package gopher

type Feature int

const (
	// Server supports those weird ASK forms from Gopher+.
	//
	// This lib is unlikely to ever support these until evidence appears of something
	// actually using them in the wild, which has so far not been forthcoming.
	FeaturePlusAsk Feature = 1

	// Server understands GopherII queries.
	FeatureII Feature = 2

	// Server will respond to GopherIIbis metadata queries.
	FeatureIIbis Feature = 3
)

type FeatureStatus int

const (
	FeatureStatusUnknown FeatureStatus = iota
	FeatureSupported
	FeatureUnsupported
)
