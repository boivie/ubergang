package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"
	"strings"
)

func (s *ApiModule) handleSigninPinQuery(w http.ResponseWriter, r *http.Request) {
	user, session, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	var req api.ApiQuerySigninPinRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	cleanPin := strings.TrimSpace(req.Pin)
	cleanPin = strings.ReplaceAll(cleanPin, "-", "")
	cleanPin = strings.ReplaceAll(cleanPin, " ", "")

	var lreq *models.SigninRequest = nil
	for _, e := range user.SigninRequests {
		if e.Pin == cleanPin {
			lreq = e
			break
		}
	}
	if lreq == nil {
		s.log.Warnf("Failed to load session: %v", err)
		jsonify(w, api.ApiQuerySigninPinResponse{
			Error: &api.ApiQuerySigninPinError{
				InvalidPin: true}})
		return
	}

	response := api.ApiQuerySigninPinResponse{
		Pin:                lreq.Pin,
		RequestorUserAgent: lreq.UserAgent,
		RequestorIP:        lreq.Ip,
		Confirmed:          lreq.Confirmed,
	}

	if !lreq.Confirmed {
		credentials := s.db.ListCredentials(user.Id)
		token, assertionRequest, err :=
			s.webauthn.CreateAssertion(user, credentials, func(state *models.AuthenticationState) {
				state.Type = &models.AuthenticationState_ConfirmSignin{
					ConfirmSignin: &models.AuthenticationStateConfirmSignin{
						SigninRequestId: lreq.Id,
						SessionId:       session.Id,
					},
				}
			})
		if err != nil {
			jsonify(w, api.ApiQuerySigninPinResponse{
				Error: &api.ApiQuerySigninPinError{
					InvalidCredentials: true}})
			return
		}

		response.Token = token
		response.AssertionRequest = assertionRequest
	}

	jsonify(w, response)
}
