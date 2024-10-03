package httpresponser

import "net/http"

type query struct {
	w    http.ResponseWriter
	req  *http.Request
	done chan struct{}
}

func newQuery(w http.ResponseWriter, req *http.Request) *query {
	return &query{
		w:    w,
		req:  req,
		done: make(chan struct{}, 1),
	}
}
