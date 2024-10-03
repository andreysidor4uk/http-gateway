package httpresponser

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type HTTPResponser struct {
	server *http.Server
	queue  chan *query
	store  *queryStore
}

func (r *HTTPResponser) ListenAndServe() {
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		slog.Info("HTTP responser Shutdown...")

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		if err := r.server.Shutdown(ctx); err != nil {
			slog.Error("HTTP responser Shutdown", "err", err)
		}
		cancel()
		close(idleConnsClosed)
	}()

	slog.Info("Start HTTP responser...")
	if err := r.server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("HTTP responser failed to start", "err", err)
		os.Exit(1)
	}

	<-idleConnsClosed
}

func (r *HTTPResponser) Response(w http.ResponseWriter, req *http.Request) {
	q := newQuery(w, req)

	slog.Debug("New query for processing",
		"id", q.id,
		"method", q.req.Method,
		"url", q.req.RequestURI)

	select {
	case <-req.Context().Done():
		slog.Error("Client closed connection without waiting for request to be processed",
			"id", q.id,
			"method", q.req.Method,
			"url", q.req.RequestURI)
	case r.queue <- q:
		slog.Debug("Query is processing",
			"id", q.id,
			"method", q.req.Method,
			"url", q.req.RequestURI)

		select {
		case <-req.Context().Done():
			slog.Error("Сlient closed connection while processing request",
				"id", q.id,
				"method", q.req.Method,
				"url", q.req.RequestURI)
		case <-q.done:
			slog.Debug("Query done",
				"id", q.id,
				"method", q.req.Method,
				"url", q.req.RequestURI)
		}
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

	if strStatus := req.Header.Get("x-req-status"); strStatus != "" {
		status, err := strconv.Atoi(strStatus)
		if err != nil {
			http.Error(w, "Bad x-req-status header.", http.StatusBadRequest)
			return
		}
		q.w.WriteHeader(status)
	}

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
