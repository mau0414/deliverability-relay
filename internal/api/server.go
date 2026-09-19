package api

import (
		// "context"
		"net/http"
	
		"github.com/mau0414/deliverability-relay/internal/queue")

type Server struct {
	mux *http.ServeMux
	queue queue.Queue
}

func NewServer(q queue.Queue) *Server {

	s := &Server{
		mux: http.NewServeMux(),
		queue: q,
	}

	s.routes()

	return s

}

func (s *Server) routes() {

	s.mux.HandleFunc("GET /health", s.handleHealth())
	s.mux.HandleFunc("POST /send", s.handleSend())
}

func (s *Server) ListenAndServe(addr string) error {

	return http.ListenAndServe(addr, s.mux)
}

