package rest

import (
	"boivie/ubergang/server/api"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleSshKeyGet(w http.ResponseWriter, r *http.Request) {
	_, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}
	shortKeyId := mux.Vars(r)["id"]

	key, err := s.db.GetSshKey(shortKeyId)
	if err != nil {
		s.log.Warnf("Failed to find ssh key: %s", shortKeyId)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	expiresAt := ""
	if key.ExpiresAt != nil {
		expiresAt = key.ExpiresAt.AsTime().Format(time.RFC3339)
	}

	jsonify(w, api.ApiSSHKey{
		ID:                key.Id,
		Name:              key.Name,
		CreatedAt:         key.CreatedAt.AsTime().Format(time.RFC3339),
		ExpiresAt:         expiresAt,
		Sha256Fingerprint: base64.RawURLEncoding.EncodeToString(key.Sha256Fingerprint),
	})
}
