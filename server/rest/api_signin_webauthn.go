package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/wa"
	"bytes"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func (s *ApiModule) handleSigninWebauthn(w http.ResponseWriter, r *http.Request) {
	respondErr := func(err api.ApiSignInWebauthnError) {
		jsonify(w, api.ApiSignInWebauthResponse{Error: &err})
	}

	var req api.ApiSignInWebauthnRequest
	err := parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	var state *models.AuthenticationState
	state, err = s.getPasswordlessState(w, r, req)
	if err != nil {
		state, err = s.db.ConsumeAuthenticationState(req.Token)
	}
	if err != nil || state.GetSignIn() == nil {
		s.log.Warn("Authentication state not found or not intended for sign-in")
		respondErr(api.ApiSignInWebauthnError{InternalError: true})
		return
	}

	user, err := s.db.GetUserById(state.UserId)
	if err != nil {
		s.log.Warnf("User not found: %s; %v", state.UserId, err)
		respondErr(api.ApiSignInWebauthnError{InvalidCredential: true})
		return
	}

	credentials := s.db.ListCredentials(user.Id)
	waUser := wa.NewUser(user, credentials)
	credential, err := s.webauthn.ValidateAssertion(&req.Credential, state, waUser)
	if err != nil {
		s.log.Warnf("WebAuthn credential doesn't validate: %v", err)
		respondErr(api.ApiSignInWebauthnError{InvalidCredential: true})
		return
	}

	session, err := s.signin(r, user)
	if err != nil {
		respondErr(api.ApiSignInWebauthnError{InvalidCredential: true})
		return
	}

	matchingCredentialId := ""
	for _, c := range credentials {
		if bytes.Equal(c.GetWebauthnCredential().GetCredentialId(), credential.ID) {
			matchingCredentialId = c.Id
		}
	}

	if matchingCredentialId == "" {
		s.log.Warnf("WebAuthn credential doesn't match any credential: %v", err)
		respondErr(api.ApiSignInWebauthnError{InvalidCredential: true})
		return
	}

	err = s.db.UpdateCredential(matchingCredentialId, func(old *models.Credential) (*models.Credential, error) {
		if old == nil {
			return nil, errors.New("credential ID collision")
		}
		old.LastUsedAt = timestamppb.New(time.Now())
		old.GetWebauthnCredential().SignCount = credential.Authenticator.SignCount
		old.GetWebauthnCredential().CloneWarning = credential.Authenticator.CloneWarning

		// Only if credential is not already associated with a session
		if !contains(old.UsedBySessionIds, session.Id) {
			old.UsedBySessionIds = append(old.UsedBySessionIds, session.Id)
		}
		return old, nil
	})
	if err != nil {
		respondErr(api.ApiSignInWebauthnError{InvalidCredential: true})
		return
	}

	jsonify(w, api.ApiSignInWebauthResponse{
		Success: &api.ApiSignInWebauthnSuccess{
			Cookie:   s.session.CreateSessionCookie(session).String(),
			Redirect: s.createRedirect(req.Redirect, session),
		}})
}

func (s *ApiModule) getPasswordlessState(w http.ResponseWriter, r *http.Request, req api.ApiSignInWebauthnRequest) (*models.AuthenticationState, error) {
	token, err := jwt.ParseWithClaims(req.Token, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(req.Credential.Response.UserHandle)
	if err != nil {
		return nil, err
	}

	return &models.AuthenticationState{
		UserId:    string(decoded),
		Challenge: claims.Subject,
		ExpiresAt: timestamppb.New(claims.ExpiresAt.Time),
		Type: &models.AuthenticationState_SignIn{
			SignIn: &models.AthenticationStateSignIn{},
		},
	}, nil
}

func (s *ApiModule) createRedirect(redirect string, session *models.Session) string {
	if redirect == "" {
		return ""
	}
	u, err := url.Parse(redirect)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("_ubergang_session", s.session.EncodeSessionCookie(session))
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *ApiModule) signin(r *http.Request, user *models.User) (session *models.Session, err error) {
	userAgent := r.Header.Get("user-agent")
	_, session, err = s.session.ReuseSession(r)
	if err != nil || session.UserId != user.Id {
		session, err = s.auth.CreateSession(user.Id, userAgent, r.RemoteAddr)
	}
	if err != nil {
		return
	}
	session.UserAgent = userAgent
	session.RemoteAddr = r.RemoteAddr

	// Update the session with new IP.
	err = s.db.UpdateSession(session.Id, func(old *models.Session) (*models.Session, error) {
		return session, nil
	})
	return
}
