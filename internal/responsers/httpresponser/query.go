package httpresponser

import (
	"net/http"

	"github.com/google/uuid"
)

type query struct {
	id   uuid.UUID
	w    http.ResponseWriter
	req  *http.Request
	done chan struct{}
}

func newQuery(w http.ResponseWriter, req *http.Request) *query {
	return &query{
		id:   uuid.New(),
		w:    w,
		req:  req,
		done: make(chan struct{}, 1),
	}
}
