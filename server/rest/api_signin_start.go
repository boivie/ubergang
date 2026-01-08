package rest

import (
	"boivie/ubergang/server/api"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/golang-jwt/jwt/v5"
)

func (s *ApiModule) handleSigninStart(w http.ResponseWriter, r *http.Request) {
	// Note: This endpoint should not be authenticated, as it's part of the
	// sign-in flow.

	challenge, err := protocol.CreateChallenge()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	challengeStr := base64.RawURLEncoding.EncodeToString(challenge)

	key := []byte("secret")
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(300 * time.Second)),
		Subject:   challengeStr,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := t.SignedString(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonify(w, api.ApiStartSigninResponse{
		Token: token,
		AssertionRequest: api.ApiAssertionRequest{
			Challenge:        challengeStr,
			Timeout:          300_000,
			RPID:             s.webauthn.RPID(),
			AllowCredentials: []api.ApiPublicKeyCredentialDescriptor{},
			UserVerification: "required",
		},
	})
}
