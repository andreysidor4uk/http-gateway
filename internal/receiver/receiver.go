package receiver

import (
	"net/http"
)

type Responser interface {
	Response(w http.ResponseWriter, req *http.Request)
}

type Receiver struct {
	server    *http.Server
	responser Responser
}

func (r *Receiver) ListenAndServe() error {
	return r.server.ListenAndServe()
}

func (r *Receiver) recive(w http.ResponseWriter, req *http.Request) {
	r.responser.Response(w, req)
}

func New(addr string, responser Responser) *Receiver {
	mux := http.NewServeMux()
	receiver := Receiver{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		responser: responser,
	}

	mux.HandleFunc("/", receiver.recive)

	return &receiver
}
