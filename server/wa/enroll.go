package wa

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func getCredentialSid(credentialID []byte) string {
	// The 144 bit truncated SHA-256 hash of the credential ID, base64 encoded.
	hash := sha256.Sum256(credentialID)
	return base64.RawURLEncoding.EncodeToString(hash[0:18])
}

func (s *WA) CreateEnrollRequest(user *models.User, sessionId string, credentials []*models.Credential) (state *models.AuthenticationState, ret *api.ApiPublicKeyCredentialCreationOptions, err error) {
	creation, session, err := s.webAuthn.BeginRegistration(NewUser(user, credentials))
	if err != nil {
		return
	}

	state = &models.AuthenticationState{
		UserVerification:   string(session.UserVerification),
		UserId:             user.Id,
		Challenge:          session.Challenge,
		AllowedCredentials: session.AllowedCredentialIDs,
		ExpiresAt:          timestamppb.New(session.Expires),
		Type: &models.AuthenticationState_Enroll{
			Enroll: &models.AuthenticationStateEnroll{
				SessionId: sessionId,
			},
		},
	}

	ret = &api.ApiPublicKeyCredentialCreationOptions{
		RP: api.ApiEnrollRP{
			Name: creation.Response.RelyingParty.Name,
			ID:   creation.Response.RelyingParty.ID,
		},
		User: api.ApiEnrollUser{
			Name:        user.DisplayName,
			ID:          base64.RawURLEncoding.EncodeToString([]byte(user.Id)),
			DisplayName: user.DisplayName,
		},
		Challenge: base64.RawURLEncoding.EncodeToString(creation.Response.Challenge),
		Parameters: lo.Map(creation.Response.Parameters, func(param protocol.CredentialParameter, _ int) api.ApiPublicKeyCredentialParameters {
			return api.ApiPublicKeyCredentialParameters{
				Type:      string(param.Type),
				Algorithm: int(param.Algorithm),
			}
		}),
		Attestation: string(creation.Response.Attestation),
		Timeout:     creation.Response.Timeout,
		ExcludeCredentials: lo.Map(creation.Response.CredentialExcludeList, func(cred protocol.CredentialDescriptor, _ int) api.ApiPublicKeyCredentialDescriptor {
			return toPublicKeyCredentialDescriptor(cred)
		}),
		AuthenticatorSelection: api.ApiAuthenticatorSelection{
			AuthenticatorAttachment: string(creation.Response.AuthenticatorSelection.AuthenticatorAttachment),
			RequireResidentKey:      creation.Response.AuthenticatorSelection.RequireResidentKey,
			ResidentKey:             string(creation.Response.AuthenticatorSelection.ResidentKey),
			UserVerification:        string(creation.Response.AuthenticatorSelection.UserVerification),
		},
	}

	return
}

func (w *WA) CreateCredential(user *models.User, sess *models.Session, state *models.AuthenticationState, attestationResponse *api.ApiAuthenticatorAttestationResponse) (credential *models.Credential, err error) {
	session := webauthn.SessionData{
		Challenge:            state.Challenge,
		UserID:               []byte(state.UserId),
		AllowedCredentialIDs: state.AllowedCredentials,
		Expires:              state.ExpiresAt.AsTime(),
		UserVerification:     protocol.UserVerificationRequirement(state.UserVerification),
		Extensions:           map[string]interface{}{},
		CredParams:           webauthn.CredentialParametersDefault(),
	}

	response := protocol.CredentialCreationResponse{}
	response.Type = "public-key"
	response.ID = attestationResponse.ID
	clientDecoded, _ := base64.RawURLEncoding.DecodeString(attestationResponse.ClientDataJSON)
	response.AttestationResponse.ClientDataJSON = protocol.URLEncodedBase64(clientDecoded)
	decoded, _ := base64.RawURLEncoding.DecodeString(attestationResponse.AttestationObject)
	response.AttestationResponse.AttestationObject = protocol.URLEncodedBase64(decoded)
	response.AttestationResponse.Transports = attestationResponse.Transports

	pcc, err := response.Parse()
	if err != nil {
		return
	}
	cred, err := w.webAuthn.CreateCredential(NewUser(user, []*models.Credential{}), session, pcc)
	if err != nil {
		return
	}

	now := time.Now()
	credential = &models.Credential{
		Id:                 getCredentialSid(cred.ID),
		UserId:             user.Id,
		Name:               w.ResolveNameFromAaGuid(cred.Authenticator.AAGUID),
		CreatedAt:          timestamppb.New(now),
		LastUsedAt:         timestamppb.New(now),
		CreatedBySessionId: sess.Id,
		UsedBySessionIds:   []string{sess.Id},
		Type: &models.Credential_WebauthnCredential{
			WebauthnCredential: &models.WebAuthnCredential{
				CredentialId: cred.ID,
				PublicKeyDer: cred.PublicKey,
				Transports: lo.Map(cred.Transport, func(t protocol.AuthenticatorTransport, _ int) string {
					return string(t)
				}),
				Aaguid:             cred.Authenticator.AAGUID,
				SignCount:          cred.Authenticator.SignCount,
				CloneWarning:       cred.Authenticator.CloneWarning,
				Attachment:         string(cred.Authenticator.Attachment),
				FlagUserPresent:    cred.Flags.UserPresent,
				FlagUserVerified:   cred.Flags.UserVerified,
				FlagBackupEligible: cred.Flags.BackupEligible,
				FlagBackupState:    cred.Flags.BackupState,
			},
		},
	}
	return
}
