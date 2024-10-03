package receiver

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Responser interface {
	Response(w http.ResponseWriter, req *http.Request)
}

type Receiver struct {
	server    *http.Server
	responser Responser
}

func (r *Receiver) ListenAndServe() {
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		slog.Info("Receiver Shutdown...")

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		if err := r.server.Shutdown(ctx); err != nil {
			slog.Error("Receiver Shutdown", "err", err)
		}
		cancel()
		close(idleConnsClosed)
	}()

	slog.Info("Start receiver responser...")
	if err := r.server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("Receiver failed to start", "err", err)
		os.Exit(1)
	}

	<-idleConnsClosed
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
