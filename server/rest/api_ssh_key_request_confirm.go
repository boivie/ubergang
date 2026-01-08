package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleGetSshKeyConfirm(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	key, err := s.db.GetSshKey(mux.Vars(r)["id"])
	if err != nil || key.UserId != user.Id {
		jsonify(w, api.ApiGetConfirmSshKeyResponse{
			Error: &api.ApiGetConfirmSshKeyError{
				InvalidKey: true,
			},
		})
		return
	}

	credentials := s.db.ListCredentials(user.Id)
	token, credentialAssertion, err :=
		s.webauthn.CreateAssertion(user, credentials, func(state *models.AuthenticationState) {
			state.Type = &models.AuthenticationState_ConfirmSshKey{
				ConfirmSshKey: &models.AthenticationStateConfirmSshKey{
					KeyId: key.Id,
				},
			}
		})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonify(w, api.ApiGetConfirmSshKeyResponse{
		Authenticate: &api.ApiGetConfirmSshKeyAuthenticate{
			KeyName:          key.Name,
			Token:            token,
			AssertionRequest: *credentialAssertion,
		},
	})
}
