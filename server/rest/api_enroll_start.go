package rest

import (
	"boivie/ubergang/server/api"
	"net/http"

	"github.com/google/uuid"
)

func (s *ApiModule) handleEnrollStart(w http.ResponseWriter, r *http.Request) {
	user, session, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	var req api.ApiStartEnrollRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	state, options, err := s.webauthn.CreateEnrollRequest(user, session.Id, s.db.ListCredentials(user.Id))
	if err != nil {
		s.log.Warn("Failed to create enroll request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stateUuid, err := uuid.NewV7()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.db.StoreAuthenticationState(&stateUuid, state)
	if err != nil {
		s.log.Warn("Failed to store authentication state")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonify(w, api.ApiStartEnrollResponse{
		EnrollRequest: &api.ApiEnrollRequest{
			Token:   stateUuid.String(),
			Options: *options,
		},
	})
}
