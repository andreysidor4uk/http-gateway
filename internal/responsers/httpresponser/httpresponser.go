package httpresponser

import "net/http"

type HTTPResponser struct {
	server *http.Server
}

func (r *HTTPResponser) ListenAndServe() error {
	return r.server.ListenAndServe()
}

func (r *HTTPResponser) requestHandler(w http.ResponseWriter, req *http.Request) {

}

func (r *HTTPResponser) responseHandler(w http.ResponseWriter, req *http.Request) {

}

func (r *HTTPResponser) Response(w http.ResponseWriter, req *http.Request) {

}

func New(addr string) *HTTPResponser {
	mux := http.NewServeMux()
	responser := HTTPResponser{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}

	mux.HandleFunc("/request", responser.requestHandler)
	mux.HandleFunc("/response", responser.responseHandler)

	return &responser
}
