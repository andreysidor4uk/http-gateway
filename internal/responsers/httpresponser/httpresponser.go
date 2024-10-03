package httpresponser

import (
	"io"
	"net/http"

	"github.com/google/uuid"
)

type HTTPResponser struct {
	server *http.Server
	queue  chan *query
	store  *queryStore
}

func (r *HTTPResponser) ListenAndServe() error {
	return r.server.ListenAndServe()
}

func (r *HTTPResponser) Response(w http.ResponseWriter, req *http.Request) {
	q := newQuery(w, req)

	select {
	case <-req.Context().Done():
	case r.queue <- q:
	}

	select {
	case <-req.Context().Done():
	case <-q.done:
	}
}

func (r *HTTPResponser) requestHandler(w http.ResponseWriter, req *http.Request) {
	select {
	case <-req.Context().Done():
	case q := <-r.queue:
		for key, headers := range q.req.Header {
			for _, value := range headers {
				w.Header().Add(key, value)
			}
		}

		w.Header().Add("x-req-id", r.store.push(q).String())
		w.Header().Add("x-req-method", q.req.Method)
		w.Header().Add("x-req-url", q.req.RequestURI)

		io.Copy(w, q.req.Body)
	}
}

func (r *HTTPResponser) responseHandler(w http.ResponseWriter, req *http.Request) {
	idStr := req.Header.Get("x-req-id")
	if idStr == "" {
		http.Error(w, "Missing x-req-id header.", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Bad x-req-id header.", http.StatusBadRequest)
		return
	}

	q := r.store.pop(id)
	if q == nil {
		http.Error(w, "Request not found or has been processed earlier.", http.StatusBadRequest)
		return
	}

	defer func() {
		q.done <- struct{}{}
	}()

	for key, headers := range req.Header {
		for _, value := range headers {
			q.w.Header().Add(key, value)
		}
	}

	io.Copy(q.w, req.Body)
}

func New(addr string) *HTTPResponser {
	mux := http.NewServeMux()
	responser := HTTPResponser{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		queue: make(chan *query),
		store: newQueryStore(),
	}

	mux.HandleFunc("/request", responser.requestHandler)
	mux.HandleFunc("/response", responser.responseHandler)

	return &responser
}
