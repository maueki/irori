package main

import (
	"github.com/flosch/pongo2"
	"github.com/microcosm-cc/bluemonday"
)

func init() {
	pongo2.RegisterFilter("sanitize", filterSanitize)
}

func filterSanitize(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	p := bluemonday.UGCPolicy()
	return pongo2.AsSafeValue(p.Sanitize(in.String())), nil
}
