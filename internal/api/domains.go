package api

import ("net/http"
		"encoding/json"
		"errors"
		"log"
	
		"github.com/mau0414/deliverability-relay/internal/domain")


func (s *Server) handleDomains() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		var req CreateDomainRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		validationResult, err := s.dnsValidator.ValidateDomain(ctx, req.Name, domain.Selector)
		
		if err != nil {

			if errors.Is(err, domain.ErrDomainNotFound) {
				http.Error(
					w,
					"domain does not exist",
					http.StatusUnprocessableEntity,
				)
				return
			} else {
				http.Error(
					w,
					"failed to validate domain",
					http.StatusInternalServerError,
				)
				return
			}
		}

		newDomain := req.ToDomainDomain(validationResult.HasSPF, validationResult.SPFRecord, validationResult.HasDKIM, validationResult.HasDMARC, validationResult.DMARCRecord)

		if err := s.domainRepository.Save(ctx, newDomain); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to save domain"})
			return
		}

		err = s.cache.SetDomain(ctx, newDomain)

		if err != nil {
			log.Printf("warning: Redis cache domain writing failed: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateDomainResponse{ID: newDomain.ID})

	}

}