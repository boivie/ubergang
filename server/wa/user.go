package wa

import (
	"boivie/ubergang/server/models"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/samber/lo"
)

type User struct {
	user        *models.User
	credentials []*models.Credential
}

func (u *User) WebAuthnID() []byte          { return []byte(u.user.Id) }
func (u *User) WebAuthnName() string        { return u.user.Email }
func (u *User) WebAuthnDisplayName() string { return u.user.Email }
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	ret := make([]webauthn.Credential, 0)
	for _, cred := range u.credentials {
		if waCred := cred.GetWebauthnCredential(); waCred != nil {
			ret = append(ret, webauthn.Credential{
				ID:        waCred.CredentialId,
				PublicKey: waCred.PublicKeyDer,
				Transport: lo.Map(waCred.Transports, func(t string, _ int) protocol.AuthenticatorTransport {
					return protocol.AuthenticatorTransport(t)
				}),
				Flags: webauthn.CredentialFlags{
					UserPresent:    waCred.FlagUserPresent,
					UserVerified:   waCred.FlagUserVerified,
					BackupEligible: waCred.FlagBackupEligible,
					BackupState:    waCred.FlagBackupState,
				},
				Authenticator: webauthn.Authenticator{
					AAGUID:       waCred.Aaguid,
					SignCount:    waCred.SignCount,
					CloneWarning: waCred.CloneWarning,
					Attachment:   protocol.AuthenticatorAttachment(waCred.Attachment),
				},
			})
		}
	}
	return ret
}
func (u *User) WebAuthnIcon() string { return "" }

func NewUser(u *models.User, credentials []*models.Credential) webauthn.User {
	return &User{
		user:        u,
		credentials: credentials,
	}
}
