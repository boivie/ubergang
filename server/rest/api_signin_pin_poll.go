package rest

import (
	"boivie/ubergang/server/api"
	"encoding/base64"
	"net/http"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

func (s *ApiModule) handleSigninPinPoll(w http.ResponseWriter, r *http.Request) {
	// Note: This endpoint should not be authenticated, as it's part of the
	// sign-in flow.

	respondErr := func(err api.ApiPollSigninPinError) {
		jsonify(w, api.ApiPollSigninPinResponse{Error: &err})
	}

	var req api.ApiPollSigninPinRequest
	err := parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	user, err := s.db.GetUserBySigninRequest(req.Id)
	if err != nil {
		s.log.Warnf("User not found for %s", req.Id)
		respondErr(api.ApiPollSigninPinError{InvalidToken: true})
		return
	}

	for _, lreq := range user.SigninRequests {
		if lreq.Id == req.Id {
			if lreq.ExpiresAt.AsTime().Before(time.Now()) {
				respondErr(api.ApiPollSigninPinError{Expired: true})
				return
			}

			if !lreq.Confirmed {
				png, err := qrcode.Encode("https://"+s.config.AdminFqdn+"/confirm/"+lreq.Pin, qrcode.Low, 256)
				if err != nil {
					respondErr(api.ApiPollSigninPinError{InternalError: true})
					return
				}
				qrCodeUri := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(png))
				jsonify(w, api.ApiPollSigninPinResponse{Pending: &api.ApiSignInPollPending{
					Pin:        lreq.Pin,
					ConfirmUrl: "https://" + s.config.AdminFqdn + "/confirm/",
					QrCodeUrl:  qrCodeUri,
				}})
				return
			}

			session, err := s.signin(r, user)
			if err != nil {
				respondErr(api.ApiPollSigninPinError{InternalError: true})
				return
			}

			jsonify(w, api.ApiPollSigninPinResponse{
				Success: &api.ApiPollSigningPinSuccess{
					Cookie:   s.session.CreateSessionCookie(session).String(),
					Redirect: s.createRedirect(req.Redirect, session),
				}})
			return
		}
	}
	respondErr(api.ApiPollSigninPinError{InvalidToken: true})
}
