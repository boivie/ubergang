package auth

import (
	"boivie/ubergang/server/models"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	apiKeyTokenPrefix = "ug_"
	apiKeyIdLength    = 16
	// How often a key's last-used time is written, to avoid a database write
	// on every request.
	apiKeyLastUsedResolution = time.Minute
)

var ErrInvalidApiKey = errors.New("invalid API key")

// CreateApiKey creates a new API key. The returned token is the only time the
// secret is available; it can't be recovered later.
func (s *Auth) CreateApiKey(name string, scope models.ApiKeyScope, createdByUserId string) (key *models.ApiKey, token string, err error) {
	// rand.Text is base32 (A-Z, 2-7), so it never contains the "_" separator.
	id := strings.ToLower(rand.Text()[:apiKeyIdLength])
	secret := strings.ToLower(rand.Text())
	hash := sha256.Sum256([]byte(secret))

	key = &models.ApiKey{
		Id:              id,
		Name:            name,
		SecretSha256:    hash[:],
		Scope:           scope,
		CreatedAt:       timestamppb.Now(),
		CreatedByUserId: createdByUserId,
	}
	err = s.db.UpdateApiKey(id, func(old *models.ApiKey) (*models.ApiKey, error) {
		if old != nil {
			return nil, errors.New("ID collision")
		}
		return key, nil
	})
	if err != nil {
		return nil, "", err
	}
	s.log.Infof("Created API key %s (%s) with scope %d", id, name, scope)
	return key, apiKeyTokenPrefix + id + "_" + secret, nil
}

// AuthenticateApiKey returns the key that `token` belongs to, and records
// that it was used.
func (s *Auth) AuthenticateApiKey(token string) (*models.ApiKey, error) {
	rest, ok := strings.CutPrefix(token, apiKeyTokenPrefix)
	if !ok {
		return nil, ErrInvalidApiKey
	}
	id, secret, ok := strings.Cut(rest, "_")
	if !ok || id == "" || secret == "" {
		return nil, ErrInvalidApiKey
	}
	key, err := s.db.GetApiKey(id)
	if err != nil {
		return nil, ErrInvalidApiKey
	}
	hash := sha256.Sum256([]byte(secret))
	if subtle.ConstantTimeCompare(hash[:], key.SecretSha256) != 1 {
		return nil, ErrInvalidApiKey
	}

	if key.LastUsedAt == nil || time.Since(key.LastUsedAt.AsTime()) >= apiKeyLastUsedResolution {
		now := timestamppb.Now()
		err = s.db.UpdateApiKey(id, func(old *models.ApiKey) (*models.ApiKey, error) {
			if old == nil {
				return nil, ErrInvalidApiKey
			}
			old.LastUsedAt = now
			return old, nil
		})
		if err != nil {
			// Most likely deleted since it was read above.
			return nil, ErrInvalidApiKey
		}
		key.LastUsedAt = now
	}
	return key, nil
}

// ApiKeyHasScope returns true if `key` grants `required`. Scopes are ordered,
// so e.g. a write key can also read.
func ApiKeyHasScope(key *models.ApiKey, required models.ApiKeyScope) bool {
	return required != models.ApiKeyScope_API_KEY_SCOPE_UNSPECIFIED && key.Scope >= required
}
