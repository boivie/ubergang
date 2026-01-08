package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"
)

func (s *ApiModule) handleSigninEmail(w http.ResponseWriter, r *http.Request) {
	// Note: This endpoint should not be authenticated, as it's part of the
	// sign-in flow.

	respondErr := func(err api.ApiSignInEmailError) {
		jsonify(w, api.ApiSigninEmailResponse{Error: &err})
	}

	var req api.ApiSignInEmailRequest
	err := parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	email := req.Email
	if email == "" {
		respondErr(api.ApiSignInEmailError{WrongEmail: true})
		return
	}

	// Don't reveal if the email was wrong.
	user, err := s.db.GetUserByEmail(req.Email)
	if err != nil {
		s.log.Infof("User not found for %s: %v", req.Email, err)
		respondErr(api.ApiSignInEmailError{WrongEmail: true})
		return
	}

	credentials := s.db.ListCredentials(user.Id)
	if len(credentials) == 0 {
		respondErr(api.ApiSignInEmailError{NoCredentials: true})
		return
	}

	token, credentialAssertion, err :=
		s.webauthn.CreateAssertion(user, credentials, func(state *models.AuthenticationState) {
			state.Type = &models.AuthenticationState_SignIn{SignIn: &models.AthenticationStateSignIn{}}
		})
	if err != nil {
		respondErr(api.ApiSignInEmailError{InternalError: true})
		return
	}

	jsonify(w, api.ApiSigninEmailResponse{
		Success: &api.ApiSignInEmailSuccess{
			Token:            token,
			AssertionRequest: *credentialAssertion,
		},
	})
}
