package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"
	"sort"
	"time"
)

func ToBackend(b *models.Backend) api.ApiBackend {
	headers := make([]api.ApiBackendHeader, 0)
	for _, h := range b.Headers {
		headers = append(headers, api.ApiBackendHeader{
			Name:  h.Name,
			Value: h.Value,
		})
	}
	createdAt := ""
	if b.CreatedAt != nil {
		createdAt = b.CreatedAt.AsTime().Format(time.RFC3339)
	}
	updatedAt := ""
	if b.UpdatedAt != nil {
		updatedAt = b.UpdatedAt.AsTime().Format(time.RFC3339)
	}

	var accessLevel string
	switch b.AccessLevel {
	case models.AccessLevel_PUBLIC:
		accessLevel = "PUBLIC"
	case models.AccessLevel_NORMAL:
		accessLevel = "NORMAL"
	default:
		accessLevel = "NORMAL"
	}

	jsScript := ""
	if b.ScriptHandler != nil {
		jsScript = b.ScriptHandler.JsScript
	}

	return api.ApiBackend{
		Fqdn:        b.Fqdn,
		UpstreamUrl: b.UpstreamUrl,
		Headers:     headers,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		AccessLevel: accessLevel,
		JsScript:    jsScript,
	}
}

func (s *ApiModule) handleBackendList(w http.ResponseWriter, r *http.Request) {
	_, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	backends := make([]api.ApiBackend, 0)
	for _, b := range s.db.ListBackends() {
		backends = append(backends, ToBackend(b))
	}
	sort.Slice(backends, func(i, j int) bool {
		return backends[i].Fqdn < backends[j].Fqdn
	})

	jsonify(w, api.ApiListBackendsResponse{
		Backends: backends,
	})
}
