package rest

import (
	"boivie/ubergang/server/api"
	"fmt"
	"log"
	"net/http"
)

func (s *ApiModule) handleTestingSetup(w http.ResponseWriter, r *http.Request) {
	if !s.config.IsInTestMode {
		s.log.Warn("Not in test mode")
		http.Error(w, "Not in test mode", http.StatusInternalServerError)
		return
	}

	err := s.db.ClearDatabase()
	if err != nil {
		s.log.Warnf("Failed to reset database: %v", err)
		http.Error(w, "Failed to reset database", http.StatusInternalServerError)
		return
	}

	_, token, err := s.auth.CreateUser("hello@example.com", "John Doe", true, nil)
	if err != nil {
		log.Fatalf("Error creating user: %v", err)
	}

	s.log.Infof("Setting up testing environment with token %s", token)

	jsonify(w, api.ApiTestingSetupResponse{
		SigninUrl: fmt.Sprintf("/signin/%s", token),
	})
}
