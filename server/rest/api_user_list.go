package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/wa"
	"encoding/base64"
	"net/http"
	"sort"
	"time"
)

func ToApiCredential(c *models.Credential) api.ApiCredential {
	return api.ApiCredential{
		ID:         c.Id,
		Name:       c.Name,
		Type:       "webauthn",
		CreatedAt:  c.CreatedAt.AsTime().Format(time.RFC3339),
		CreatedBy:  c.CreatedBySessionId,
		LastUsedAt: c.LastUsedAt.AsTime().Format(time.RFC3339),
		UsedBy:     c.UsedBySessionIds,
		Transports: c.GetWebauthnCredential().Transports,
		Aaguid:     wa.FormatAaguidBytesToString(c.GetWebauthnCredential().Aaguid),
	}
}

func ToApiSession(s *models.Session) api.ApiSession {
	return api.ApiSession{
		ID:         s.Id,
		UserAgent:  s.UserAgent,
		RemoteAddr: s.RemoteAddr,
		CreatedAt:  s.CreatedAt.AsTime().Format(time.RFC3339),
		AccessedAt: "",
	}
}

func ToApiSshKey(key *models.SshKey) api.ApiSSHKey {
	obj := api.ApiSSHKey{
		ID:                key.Id,
		Name:              key.Name,
		CreatedAt:         key.CreatedAt.AsTime().Format(time.RFC3339),
		Sha256Fingerprint: base64.RawURLEncoding.EncodeToString(key.Sha256Fingerprint),
	}
	if key.ExpiresAt != nil {
		obj.ExpiresAt = key.ExpiresAt.AsTime().Format(time.RFC3339)
	}

	return obj
}

func (s *ApiModule) handleUserList(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	users := make([]api.ApiUser, 0)
	for _, u := range s.db.ListUsers() {
		au := api.ApiUser{
			ID:             u.Id,
			Email:          u.Email,
			DisplayName:    u.DisplayName,
			AllowedHosts:   u.AllowedHosts,
			IsAdmin:        u.IsAdmin,
			Credentials:    make([]api.ApiCredential, 0),
			Sessions:       make([]api.ApiSession, 0),
			SSHKeys:        make([]api.ApiSSHKey, 0),
			CurrentSession: nil,
		}
		for _, c := range s.db.ListCredentials(u.Id) {
			au.Credentials = append(au.Credentials, ToApiCredential(c))
		}
		for _, s := range s.db.ListSessions(u.Id) {
			au.Sessions = append(au.Sessions, ToApiSession(s))
		}
		for _, key := range s.db.ListSshKeys(u.Id) {
			au.SSHKeys = append(au.SSHKeys, ToApiSshKey(key))
		}
		users = append(users, au)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].Email < users[j].Email
	})

	jsonify(w, api.ApiListUsersResponse{
		Users: users,
	})
}
