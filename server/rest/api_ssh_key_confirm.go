package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/wa"
	"net/http"
	"time"
)

func (s *ApiModule) handlePostSshKeyConfirm(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	var req api.ApiPostConfirmSshKeyRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	state, err := s.db.ConsumeAuthenticationState(req.Token)
	if err != nil || state.GetConfirmSshKey() == nil {
		s.log.Warnf("Token not found or not intended for this user or signing type")
		jsonify(w, api.ApiPostConfirmSshKeyResponse{
			Error: &api.ApiPostConfirmSshKeyError{
				FailedAuthentication: true}})
		return
	}

	_, err = s.webauthn.ValidateAssertion(&req.Credential, state, wa.NewUser(user, s.db.ListCredentials(user.Id)))
	if err != nil {
		s.log.Warnf("Failed to validate assertion: %v", err)
		jsonify(w, api.ApiPostConfirmSshKeyResponse{
			Error: &api.ApiPostConfirmSshKeyError{
				FailedAuthentication: true}})
		return
	}

	now := time.Now()
	key, err := s.auth.ConfirmSshKeyUpdate(state.GetConfirmSshKey().KeyId, now)
	if err != nil {
		s.log.Warnf("Failed to update ssh key: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	jsonify(w, api.ApiPostConfirmSshKeyResponse{
		Result: &api.ApiPostConfirmSshKeyResult{
			ExpiresAt: key.ExpiresAt.AsTime().Format(time.RFC3339)}})
}
