package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/wa"
	"errors"
	"net/http"
)

func (s *ApiModule) handleSigninPinConfirm(w http.ResponseWriter, r *http.Request) {
	user, session, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	var req api.ApiConfirmSigninPinRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	state, err := s.db.ConsumeAuthenticationState(req.Token)
	if err != nil || state.GetConfirmSignin() == nil {
		jsonify(w, api.ApiConfirmSigninPinResponse{
			Error: &api.ApiConfirmSigninPinError{InvalidEnrollment: true}})
		return
	}

	if state.UserId != user.Id ||
		state.GetConfirmSignin().SessionId != session.Id {
		s.log.Warnf("Token not intended for this user or signing type")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	_, err = s.webauthn.ValidateAssertion(&req.Credential, state, wa.NewUser(user, s.db.ListCredentials(user.Id)))
	if err != nil {
		s.log.Warnf("Failed to validate assertion: %v", err)
		jsonify(w, api.ApiConfirmSigninPinResponse{
			Error: &api.ApiConfirmSigninPinError{InvalidEnrollment: true}})
		return
	}

	err = s.db.UpdateUser(user.Id, func(old *models.User) (*models.User, error) {
		for idx := range old.SigninRequests {
			if old.SigninRequests[idx].Id == state.GetConfirmSignin().SigninRequestId {
				old.SigninRequests[idx].Confirmed = true
				return old, nil
			}
		}
		return nil, errors.New("signin request not found")
	})
	if err != nil {
		s.log.Warnf("Failed to update user: %v", err)
		jsonify(w, api.ApiConfirmSigninPinResponse{
			Error: &api.ApiConfirmSigninPinError{InvalidEnrollment: true}})
		return
	}

	jsonify(w, api.ApiConfirmSigninPinResponse{})
}
