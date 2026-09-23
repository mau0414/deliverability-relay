package api

import (
	"encoding/json"
	"net/http"
	"log"
	"errors"

	"github.com/mau0414/deliverability-relay/internal/dns"
	"github.com/mau0414/deliverability-relay/internal/domain"
	"github.com/jackc/pgx/v5"
)


func (s *Server) handleSend() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		var req SendEmailRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		email := req.ToEmailDomain()

		if err := email.Validate(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}


		if err := s.emailRepository.Save(ctx, email); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to save email"})
			return
		}

		fromDomain, err := dns.ParseReceiverDomain(email.From)

		// if parser returns error
		if err != nil {
			s.emailRepository.UpdateStatus(ctx, email, domain.StatusFailed)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "from sent has not an valid format"})
			return
		}

		registeredDomain, err := s.cache.GetDomain(ctx, "domain:" + fromDomain)

		if err != nil {
			log.Printf("redis lookup failed for domain %s: %v", fromDomain, err)
			registeredDomain = nil // check postgre
		}

		if registeredDomain == nil {
			registeredDomain, err = s.domainRepository.FindDomain(ctx, fromDomain)

			if err != nil {

				if errors.Is(err, pgx.ErrNoRows) {

					s.emailRepository.UpdateStatus(ctx, email, domain.StatusFailed)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnprocessableEntity)
					json.NewEncoder(w).Encode(map[string]string{"error": "sender domain is not registered"})
					return

				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "failed to verify sender domain"})
				return
			}

			if err := s.cache.SetDomain(ctx, *registeredDomain); err != nil {
				log.Printf("failed to populate redis cache for domain %s: %v", fromDomain, err)
			}

		} 

		if !registeredDomain.HasSPF || !registeredDomain.HasDKIM {
			s.emailRepository.UpdateStatus(ctx, email, domain.StatusFailed)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"error": "email domain has no SPF and/or DKIM record"})
			return
		}



		if err := s.queue.Enqueue(email); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}


		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(SendEmailResponse{ID: email.ID})

	}

}