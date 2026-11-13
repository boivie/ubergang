package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const maxApiKeyNameLength = 100

func apiKeyScopeName(scope models.ApiKeyScope) string {
	switch scope {
	case models.ApiKeyScope_API_KEY_SCOPE_READ:
		return "read"
	case models.ApiKeyScope_API_KEY_SCOPE_WRITE:
		return "write"
	case models.ApiKeyScope_API_KEY_SCOPE_ADMIN:
		return "admin"
	}
	return ""
}

// parseApiKeyScope parses the scopes that keys can be created with.
func parseApiKeyScope(name string) (models.ApiKeyScope, bool) {
	switch name {
	case "read":
		return models.ApiKeyScope_API_KEY_SCOPE_READ, true
	case "write":
		return models.ApiKeyScope_API_KEY_SCOPE_WRITE, true
	case "admin":
		return models.ApiKeyScope_API_KEY_SCOPE_ADMIN, true
	}
	return models.ApiKeyScope_API_KEY_SCOPE_UNSPECIFIED, false
}

func (s *ApiModule) toApiApiKey(key *models.ApiKey) api.ApiApiKey {
	ret := api.ApiApiKey{
		Id:        key.Id,
		Name:      key.Name,
		Scope:     apiKeyScopeName(key.Scope),
		CreatedAt: key.CreatedAt.AsTime().Format(time.RFC3339),
	}
	if user, err := s.db.GetUserById(key.CreatedByUserId); err == nil {
		ret.CreatedBy = user.Email
	}
	if key.LastUsedAt != nil {
		ret.LastUsedAt = key.LastUsedAt.AsTime().Format(time.RFC3339)
	}
	return ret
}

// requireAdmin returns the signed-in user if they're an admin, and otherwise
// writes an error response and returns nil.
func (s *ApiModule) requireAdmin(w http.ResponseWriter, r *http.Request) *models.User {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return nil
	}
	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return nil
	}
	return user
}

func (s *ApiModule) handleApiKeyList(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}

	keys := make([]api.ApiApiKey, 0)
	for _, key := range s.db.ListApiKeys() {
		keys = append(keys, s.toApiApiKey(key))
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].CreatedAt < keys[j].CreatedAt
	})
	jsonify(w, api.ApiListApiKeysResponse{ApiKeys: keys})
}

func (s *ApiModule) handleApiKeyCreate(w http.ResponseWriter, r *http.Request) {
	user := s.requireAdmin(w, r)
	if user == nil {
		return
	}

	var req api.ApiCreateApiKeyRequest
	if err := parseJsonRequest(w, r, &req); err != nil {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > maxApiKeyNameLength {
		http.Error(w, "Invalid name", http.StatusBadRequest)
		return
	}
	scope, ok := parseApiKeyScope(req.Scope)
	if !ok {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	key, token, err := s.auth.CreateApiKey(name, scope, user.Id)
	if err != nil {
		s.log.Warnf("Failed to create API key: %v", err)
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}
	jsonify(w, api.ApiCreateApiKeyResponse{
		ApiKey: s.toApiApiKey(key),
		Token:  token,
	})
}

func (s *ApiModule) handleApiKeyDelete(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}

	id := mux.Vars(r)["id"]
	errNotFound := errors.New("not found")
	err := s.db.UpdateApiKey(id, func(old *models.ApiKey) (*models.ApiKey, error) {
		if old == nil {
			return nil, errNotFound
		}
		return nil, nil
	})
	if errors.Is(err, errNotFound) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to delete API key", http.StatusInternalServerError)
		return
	}
	s.log.Infof("Deleted API key %s", id)
	w.WriteHeader(http.StatusNoContent)
}
