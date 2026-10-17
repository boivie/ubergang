package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"
)

func (s *ApiModule) handleSigninStart(w http.ResponseWriter, r *http.Request) {
	// Note: This endpoint should not be authenticated, as it's part of the
	// sign-in flow.

	token, assertionRequest, err :=
		s.webauthn.CreateDiscoverableAssertion(func(state *models.AuthenticationState) {
			state.Type = &models.AuthenticationState_SignIn{SignIn: &models.AthenticationStateSignIn{}}
		})
	if err != nil {
		s.log.Warnf("Failed to create assertion: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonify(w, api.ApiStartSigninResponse{
		Token:            token,
		AssertionRequest: *assertionRequest,
	})
}
