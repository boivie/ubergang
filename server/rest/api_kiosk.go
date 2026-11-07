package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/auth"
	"boivie/ubergang/server/common"
	"boivie/ubergang/server/kiosk"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/wa"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

const (
	// The longest that a kiosk session can be valid for, when it expires.
	maxKioskSessionDuration   = 10 * 365 * 24 * time.Hour
	maxKioskSessionNameLength = 100
)

// kioskRedirect parses `redirect`, and returns it and its host if it's a URL
// on a backend that allows kiosk sign-in.
func (s *ApiModule) kioskRedirect(redirect string) (*url.URL, string, error) {
	u, err := url.Parse(redirect)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Hostname() == "" {
		return nil, "", fmt.Errorf("invalid redirect %q", redirect)
	}
	host := strings.ToLower(u.Hostname())
	adminHost := s.config.AdminFqdn
	if h, _, err := net.SplitHostPort(adminHost); err == nil {
		adminHost = h
	}
	if host == strings.ToLower(adminHost) {
		return nil, "", errors.New("kiosk sessions can't be used on the admin host")
	}
	backend, err := s.db.GetBackend(host)
	if err != nil {
		return nil, "", fmt.Errorf("no backend for %s", host)
	}
	if !backend.AllowKioskSignin {
		return nil, "", fmt.Errorf("kiosk sign-in isn't allowed for %s", host)
	}
	return u, host, nil
}

func (s *ApiModule) kioskConfirmUrl() string {
	return "https://" + s.config.AdminFqdn + "/kiosk/"
}

func (s *ApiModule) handleKioskRequest(w http.ResponseWriter, r *http.Request) {
	// Note: This endpoint should not be authenticated, as it's part of the
	// sign-in flow.

	respondErr := func(err api.ApiRequestKioskSigninError) {
		jsonify(w, api.ApiRequestKioskSigninResponse{Error: &err})
	}

	var req api.ApiRequestKioskSigninRequest
	if err := parseJsonRequest(w, r, &req); err != nil {
		return
	}

	redirect, host, err := s.kioskRedirect(req.Redirect)
	if err != nil {
		s.log.Infof("Not offering kiosk sign-in: %v", err)
		respondErr(api.ApiRequestKioskSigninError{NotAllowed: true})
		return
	}

	ip := common.ReadUserIP(r, s.trustProxyHeaders)
	kr, err := s.kiosk.Create(host, redirect.String(), ip, r.UserAgent())
	if errors.Is(err, kiosk.ErrTooManyRequests) {
		s.log.Warnf("Too many kiosk sign-in requests (from %s)", ip)
		respondErr(api.ApiRequestKioskSigninError{TooManyRequests: true})
		return
	}
	if err != nil {
		s.log.Errorf("Failed to create kiosk sign-in request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	png, err := qrcode.Encode(s.kioskConfirmUrl()+kr.Code, qrcode.Low, 256)
	if err != nil {
		s.log.Errorf("Failed to create QR code: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonify(w, api.ApiRequestKioskSigninResponse{
		Success: &api.ApiRequestKioskSigninSuccess{
			ID:         kr.ID,
			Code:       kr.Code,
			ConfirmUrl: s.kioskConfirmUrl(),
			QrCodeUrl:  "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
			ExpiresAt:  kr.ExpiresAt.Format(time.RFC3339),
		},
	})
}

func (s *ApiModule) handleKioskPoll(w http.ResponseWriter, r *http.Request) {
	// Note: This endpoint should not be authenticated, as it's part of the
	// sign-in flow.

	var req api.ApiPollKioskSigninRequest
	if err := parseJsonRequest(w, r, &req); err != nil {
		return
	}

	kr, err := s.kiosk.Get(req.ID)
	if err != nil {
		jsonify(w, api.ApiPollKioskSigninResponse{Expired: true})
		return
	}
	if kr.Approval == nil {
		jsonify(w, api.ApiPollKioskSigninResponse{Pending: true})
		return
	}

	// Only one poll gets the session.
	kr, err = s.kiosk.TakeApproved(req.ID)
	if err != nil {
		jsonify(w, api.ApiPollKioskSigninResponse{Expired: true})
		return
	}
	redirect, err := s.completeKioskSignin(kr)
	if err != nil {
		// The kiosk will make a new request.
		s.log.Warnf("Failed to complete kiosk sign-in: %v", err)
		jsonify(w, api.ApiPollKioskSigninResponse{Expired: true})
		return
	}
	jsonify(w, api.ApiPollKioskSigninResponse{Redirect: redirect})
}

// completeKioskSignin creates the session for an approved request, and
// returns the URL that sets it on the kiosk.
func (s *ApiModule) completeKioskSignin(kr kiosk.Request) (string, error) {
	// Check again, in case something changed since the request was approved.
	redirect, host, err := s.kioskRedirect(kr.Redirect)
	if err != nil {
		return "", err
	}
	user, err := s.db.GetUserById(kr.Approval.UserID)
	if err != nil {
		return "", err
	}
	if !auth.CanAccessHost(user, host) {
		return "", fmt.Errorf("%s can't access %s", user.Email, host)
	}

	var expiresAt time.Time
	if kr.Approval.Duration > 0 {
		expiresAt = time.Now().Add(kr.Approval.Duration)
	}
	session, err := s.auth.CreateKioskSession(user.Id, kr.UserAgent, kr.IP, host, kr.Approval.Name, expiresAt)
	if err != nil {
		return "", err
	}
	s.log.Infof("Signed in kiosk at %s to %s as %s (session %s)", kr.IP, host, user.Email, session.Id)
	return s.session.TrampolineURL(session, redirect)
}

func (s *ApiModule) handleKioskQuery(w http.ResponseWriter, r *http.Request) {
	user, session, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	respondErr := func(err api.ApiQueryKioskSigninError) {
		jsonify(w, api.ApiQueryKioskSigninResponse{Error: &err})
	}

	var req api.ApiQueryKioskSigninRequest
	if err := parseJsonRequest(w, r, &req); err != nil {
		return
	}

	code := strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(req.Code))
	kr, err := s.kiosk.GetPendingByCode(code, user.Id)
	if errors.Is(err, kiosk.ErrTooManyRequests) {
		s.log.Warnf("Too many invalid kiosk codes entered by %s", user.Email)
		respondErr(api.ApiQueryKioskSigninError{TooManyAttempts: true})
		return
	}
	if err != nil {
		respondErr(api.ApiQueryKioskSigninError{InvalidCode: true})
		return
	}
	if !auth.CanAccessHost(user, kr.Host) {
		respondErr(api.ApiQueryKioskSigninError{NotAllowed: true})
		return
	}

	credentials := s.db.ListCredentials(user.Id)
	if len(credentials) == 0 {
		respondErr(api.ApiQueryKioskSigninError{NoCredentials: true})
		return
	}
	token, assertionRequest, err :=
		s.webauthn.CreateAssertion(user, credentials, func(state *models.AuthenticationState) {
			state.Type = &models.AuthenticationState_ApproveKiosk{
				ApproveKiosk: &models.AuthenticationStateApproveKiosk{
					KioskRequestId: kr.ID,
					SessionId:      session.Id,
				},
			}
		})
	if err != nil {
		s.log.Warnf("Failed to create assertion: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	approverIP := common.ReadUserIP(r, s.trustProxyHeaders)
	jsonify(w, api.ApiQueryKioskSigninResponse{
		Host:               kr.Host,
		RequestorIP:        kr.IP,
		RequestorUserAgent: kr.UserAgent,
		RequestedAt:        kr.CreatedAt.Format(time.RFC3339),
		ApproverIP:         approverIP,
		SameNetwork:        kiosk.SameNetwork(kr.IP, approverIP),
		Token:              token,
		AssertionRequest:   assertionRequest,
	})
}

func (s *ApiModule) handleKioskApprove(w http.ResponseWriter, r *http.Request) {
	user, session, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	respondErr := func(err api.ApiApproveKioskSigninError) {
		jsonify(w, api.ApiApproveKioskSigninResponse{Error: &err})
	}

	var req api.ApiApproveKioskSigninRequest
	if err := parseJsonRequest(w, r, &req); err != nil {
		return
	}

	state, err := s.db.ConsumeAuthenticationState(req.Token)
	if err != nil || state.GetApproveKiosk() == nil {
		respondErr(api.ApiApproveKioskSigninError{InvalidRequest: true})
		return
	}
	if state.UserId != user.Id || state.GetApproveKiosk().SessionId != session.Id {
		s.log.Warnf("Kiosk approval not intended for this user or session")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	duration := time.Duration(req.DurationSeconds) * time.Second
	name := strings.TrimSpace(req.Name)
	if req.DurationSeconds < 0 || duration > maxKioskSessionDuration ||
		len(name) > maxKioskSessionNameLength {
		respondErr(api.ApiApproveKioskSigninError{InvalidRequest: true})
		return
	}

	_, err = s.webauthn.ValidateAssertion(&req.Credential, state, wa.NewUser(user, s.db.ListCredentials(user.Id)))
	if err != nil {
		s.log.Warnf("Failed to validate assertion: %v", err)
		respondErr(api.ApiApproveKioskSigninError{InvalidCredential: true})
		return
	}

	err = s.kiosk.Approve(state.GetApproveKiosk().KioskRequestId, kiosk.Approval{
		UserID:   user.Id,
		Name:     name,
		Duration: duration,
	})
	if err != nil {
		s.log.Infof("Failed to approve kiosk sign-in: %v", err)
		respondErr(api.ApiApproveKioskSigninError{Expired: true})
		return
	}
	jsonify(w, api.ApiApproveKioskSigninResponse{})
}
