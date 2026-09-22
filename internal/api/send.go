package api

import ("net/http"
		"encoding/json")


func (s *Server) handleSend() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

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

		// todo pre-flight com cache depois postgre

		if err := s.emailRepository.Save(r.Context(), email); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to save email"})
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