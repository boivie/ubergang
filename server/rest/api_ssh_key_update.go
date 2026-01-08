package rest

import (
	"boivie/ubergang/server/api"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleUpdateSshKey(w http.ResponseWriter, r *http.Request) {
	var req api.ApiProposeSshKeyRequest
	err := parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	keyID := mux.Vars(r)["id"]
	key, err := s.auth.UpdateSshKey(keyID, req.KeySecret, req.PublicKey)
	if err != nil {
		s.log.Warnf("Failed to update key: %v", err)
		jsonify(w, api.ApiProposeSshKeyResponse{})
		return
	}

	url := fmt.Sprintf("https://%s/ssh/%s", s.config.AdminFqdn, key.Id)
	jsonify(w, api.ApiProposeSshKeyResponse{
		ConfirmUrl: url,
	})
}
