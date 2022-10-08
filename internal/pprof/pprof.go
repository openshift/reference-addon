package pprof

import (
	"context"
	"net/http"
	"net/http/pprof"
)

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return &Server{
		s: &http.Server{Addr: addr, Handler: mux},
	}
}

type Server struct {
	s *http.Server
}

func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error)
	drain := func() {
		for range errCh {
		}
	}
	defer drain()

	go func() {
		defer close(errCh)

		errCh <- s.s.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.s.Close()
	}
}
