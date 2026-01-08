package wa

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"encoding/base64"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toPublicKeyCredentialDescriptor(cred protocol.CredentialDescriptor) api.ApiPublicKeyCredentialDescriptor {
	return api.ApiPublicKeyCredentialDescriptor{
		Type: string(cred.Type),
		ID:   base64.RawURLEncoding.EncodeToString(cred.CredentialID),
		Transports: lo.Map(cred.Transport, func(t protocol.AuthenticatorTransport, _ int) string {
			return string(t)
		}),
	}
}

func (s *WA) CreateAssertion(user *models.User, credentials []*models.Credential, fixupState func(state *models.AuthenticationState)) (token string, credentialAssertion *api.ApiAssertionRequest, err error) {
	options, st, err := s.webAuthn.BeginLogin(NewUser(user, credentials))
	if err != nil {
		return
	}

	if string(st.UserID) != user.Id {
		err = fmt.Errorf("internal error (user mismatch)")
		return
	}

	stateUuid, err := uuid.NewV7()
	if err != nil {
		return
	}
	token = stateUuid.String()

	state := &models.AuthenticationState{
		UserVerification:   string(st.UserVerification),
		UserId:             user.Id,
		Challenge:          st.Challenge,
		AllowedCredentials: st.AllowedCredentialIDs,
		ExpiresAt:          timestamppb.New(st.Expires)}
	fixupState(state)
	err = s.db.StoreAuthenticationState(&stateUuid, state)
	if err != nil {
		return
	}

	credentialAssertion = &api.ApiAssertionRequest{
		Challenge: base64.RawURLEncoding.EncodeToString(options.Response.Challenge),
		Timeout:   options.Response.Timeout,
		RPID:      options.Response.RelyingPartyID,
		AllowCredentials: lo.Map(options.Response.AllowedCredentials, func(cred protocol.CredentialDescriptor, _ int) api.ApiPublicKeyCredentialDescriptor {
			return toPublicKeyCredentialDescriptor(cred)
		}),
		UserVerification: string(options.Response.UserVerification),
	}
	return
}

func (s *WA) ValidateAssertion(cred *api.ApiAssertionCredential, state *models.AuthenticationState, user webauthn.User) (*webauthn.Credential, error) {
	st := webauthn.SessionData{
		UserVerification:     protocol.UserVerificationRequirement(state.UserVerification),
		Challenge:            state.Challenge,
		UserID:               []byte(state.UserId),
		AllowedCredentialIDs: state.AllowedCredentials,
		Expires:              state.ExpiresAt.AsTime(),
		Extensions:           map[string]interface{}{},
	}

	var decoded []byte
	cat := protocol.CredentialAssertionResponse{}
	decoded, _ = base64.RawURLEncoding.DecodeString(cred.ID)
	cat.RawID = decoded
	cat.ID = cred.ID
	cat.Type = "public-key"
	decoded, _ = base64.RawURLEncoding.DecodeString(cred.Response.ClientDataJSON)
	cat.AssertionResponse.ClientDataJSON = decoded
	decoded, _ = base64.RawURLEncoding.DecodeString(cred.Response.AuthenticatorData)
	cat.AssertionResponse.AuthenticatorData = decoded
	decoded, _ = base64.RawURLEncoding.DecodeString(cred.Response.Signature)
	cat.AssertionResponse.Signature = decoded
	decoded, _ = base64.RawURLEncoding.DecodeString(cred.Response.UserHandle)
	cat.AssertionResponse.UserHandle = decoded
	par, err := cat.Parse()
	if err != nil {
		return nil, err
	}

	return s.webAuthn.ValidateLogin(user, st, par)
}
