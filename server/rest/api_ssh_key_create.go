package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
)

func (s *ApiModule) handleSshKeyCreate(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	var req api.ApiCreateSshKeyRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	key, err := s.auth.CreateSshKey(user.Id, req.Name)
	if err != nil {
		s.log.Warnf("Failed to create key: %v", err)
		return
	}
	jsonify(w, api.ApiCreateSshKeyResponse{
		KeyID: key.Id,
	})
}
