package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"
)

func (s *ApiModule) handleBootstrapStatus(w http.ResponseWriter, r *http.Request) {
	// Check if server is configured
	isConfigured := s.config.Email != "" && s.config.SiteFqdn != "" && s.config.AdminFqdn != ""

	jsonify(w, api.ApiBootstrapStatusResponse{
		IsConfigured: isConfigured,
	})
}

func (s *ApiModule) handleBootstrapConfigure(w http.ResponseWriter, r *http.Request) {
	// Parse the request
	var req api.ApiBootstrapConfigureRequest
	err := parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	// Validate input
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	if req.SiteFqdn == "" {
		http.Error(w, "Site FQDN is required", http.StatusBadRequest)
		return
	}

	// Check if already configured
	isConfigured := s.config.Email != "" && s.config.SiteFqdn != "" && s.config.AdminFqdn != ""
	if isConfigured {
		http.Error(w, "Server is already configured", http.StatusBadRequest)
		return
	}

	// Save configuration
	adminFqdn := "account." + req.SiteFqdn
	err = s.db.UpdateConfiguration(func(old *models.Configuration) (*models.Configuration, error) {
		if old == nil {
			old = &models.Configuration{}
		}
		old.Email = req.Email
		old.SiteFqdn = req.SiteFqdn
		old.AdminFqdn = adminFqdn
		return old, nil
	})

	if err != nil {
		s.log.Errorf("Failed to save configuration: %v", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	s.log.Infof("Bootstrap configuration saved: email=%s, siteFqdn=%s, adminFqdn=%s",
		req.Email, req.SiteFqdn, adminFqdn)

	jsonify(w, api.ApiBootstrapConfigureResponse{
		AdminFqdn: adminFqdn,
	})
}
