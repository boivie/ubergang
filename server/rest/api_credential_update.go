package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleCredentialUpdate(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	var req api.ApiUpdateCredentialRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	credId := mux.Vars(r)["id"]

	err = s.db.UpdateCredential(credId, func(old *models.Credential) (*models.Credential, error) {
		if old == nil {
			return nil, errors.New("credential not found")
		}
		// Ensure the credential belongs to the currently authenticated user.
		if old.UserId != user.Id {
			// Return the same error to avoid leaking information about credential existence.
			return nil, errors.New("credential not found")
		}
		if req.Name != nil {
			old.Name = *req.Name
		}
		return old, nil
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	jsonify(w, api.ApiUpdateCredentialResponse{})
}
