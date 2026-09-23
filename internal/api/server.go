package api

import ("net/http"
	
		"github.com/mau0414/deliverability-relay/internal/queue"
		"github.com/mau0414/deliverability-relay/internal/dns"
		"github.com/mau0414/deliverability-relay/internal/repository"
		"github.com/mau0414/deliverability-relay/internal/cache")

type Server struct {
	mux *http.ServeMux
	queue queue.Queue
	emailRepository *repository.EmailRepository
	domainRepository *repository.DomainRepository
	dnsValidator *dns.Validator
	cache *cache.Redis
	skipPreFlight bool
}

func NewServer(q queue.Queue, emailRepository *repository.EmailRepository, domainRepository *repository.DomainRepository, dnsValidator *dns.Validator, redis *cache.Redis, skipPreFlight bool) *Server {

	s := &Server{
		mux: http.NewServeMux(),
		queue: q,
		emailRepository: emailRepository,
		domainRepository: domainRepository,
		dnsValidator: dnsValidator,
		cache: redis,
		skipPreFlight: skipPreFlight, 
	}

	s.routes()

	return s

}

func (s *Server) routes() {

	s.mux.HandleFunc("GET /health", s.handleHealth())
	s.mux.HandleFunc("POST /domains", s.handleDomains())
	s.mux.HandleFunc("POST /send", s.handleSend())
	s.mux.HandleFunc("GET /dashboard", s.handleDashboard())
}

func (s *Server) ListenAndServe(addr string) error {

	return http.ListenAndServe(addr, s.mux)
}

