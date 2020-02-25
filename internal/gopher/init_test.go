package gopher

func mustParseURL(s string) URL {
	gu, err := ParseURL(s)
	if err != nil {
		panic(err)
	}
	return gu
}
