package api

import ("net/http"
	
		"github.com/mau0414/deliverability-relay/internal/queue"
		"github.com/mau0414/deliverability-relay/internal/repository")

type Server struct {
	mux *http.ServeMux
	queue queue.Queue
	repository *repository.EmailRepository
}

func NewServer(q queue.Queue, r *repository.EmailRepository) *Server {

	s := &Server{
		mux: http.NewServeMux(),
		queue: q,
		repository: r,
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

