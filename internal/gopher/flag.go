package gopher

type URLVar URL

func (uv URLVar) URL() URL {
	return URL(uv)
}

func (uv URLVar) String() string {
	return URL(uv).String()
}

func (uv *URLVar) Set(s string) (err error) {
	u, err := ParseURL(s)
	if err != nil {
		return err
	}
	*uv = URLVar(u)
	return nil
}
