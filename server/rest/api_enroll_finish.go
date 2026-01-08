package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"

	"errors"
	"net/http"
)

func (s *ApiModule) handleEnrollFinish(w http.ResponseWriter, r *http.Request) {
	user, session, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}
	var req api.ApiFinishEnrollRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	state, err := s.db.ConsumeAuthenticationState(req.Token)
	if err != nil || state.GetEnroll() == nil {
		jsonify(w, api.ApiFinishEnrollResponse{
			Error: &api.ApiFinishEnrollError{InvalidEnrollment: true}})
		return
	}

	if state.GetEnroll().SessionId != session.Id {
		jsonify(w, api.ApiFinishEnrollResponse{
			Error: &api.ApiFinishEnrollError{InvalidEnrollment: true}})
		return
	}

	cred, err := s.webauthn.CreateCredential(user, session, state, &req.AttestationResponse)
	if err != nil {
		jsonify(w, api.ApiFinishEnrollResponse{
			Error: &api.ApiFinishEnrollError{InvalidEnrollment: true}})
		return
	}

	err = s.db.UpdateCredential(cred.Id, func(old *models.Credential) (*models.Credential, error) {
		if old != nil {
			return nil, errors.New("credential ID collision")
		}
		return cred, nil
	})
	if err != nil {
		return
	}

	apiCredential := ToApiCredential(cred)
	jsonify(w, api.ApiFinishEnrollResponse{Credential: &apiCredential})
}
