package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/auth"
	"boivie/ubergang/server/models"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gorilla/mux"
)

// The public API is authenticated only with API keys sent as bearer tokens.
// Session cookies are never accepted, which also makes it immune to CSRF.

const (
	publicApiPrefix     = "/api/v1"
	maxPublicBodyBytes  = 1 << 20
	maxPublicIdLength   = 256
	publicErrBadRequest = "invalid_request"
	publicErrNotFound   = "not_found"
	publicErrConflict   = "conflict"
	publicErrInternal   = "internal_error"
)

type publicHandlerFunc func(w http.ResponseWriter, r *http.Request, key *models.ApiKey)

func (s *ApiModule) registerPublicEndpoints(r *mux.Router) {
	v1 := r.Host(s.config.AdminFqdn).PathPrefix(publicApiPrefix).Subrouter()
	// Also used for unsupported methods, as mux's MethodNotAllowedHandler
	// doesn't trigger for routes with a Host matcher.
	v1.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writePublicError(w, http.StatusNotFound, publicErrNotFound, "No such endpoint")
	})

	read := models.ApiKeyScope_API_KEY_SCOPE_READ
	write := models.ApiKeyScope_API_KEY_SCOPE_WRITE
	admin := models.ApiKeyScope_API_KEY_SCOPE_ADMIN

	route := func(method, path string, scope models.ApiKeyScope, h publicHandlerFunc) {
		v1.Methods(method).Path(path).HandlerFunc(s.withApiKey(scope, h))
	}

	route("GET", "/mqtt/profiles", read, s.handlePublicMqttProfileList)
	route("POST", "/mqtt/profiles", write, s.handlePublicMqttProfileCreate)
	route("GET", "/mqtt/profiles/{id}", read, s.handlePublicMqttProfileGet)
	route("PATCH", "/mqtt/profiles/{id}", write, s.handlePublicMqttProfileUpdate)
	route("DELETE", "/mqtt/profiles/{id}", write, s.handlePublicMqttProfileDelete)

	route("GET", "/mqtt/clients", read, s.handlePublicMqttClientList)
	route("POST", "/mqtt/clients", write, s.handlePublicMqttClientCreate)
	route("GET", "/mqtt/clients/{id}", read, s.handlePublicMqttClientGet)
	route("PATCH", "/mqtt/clients/{id}", write, s.handlePublicMqttClientUpdate)
	route("DELETE", "/mqtt/clients/{id}", write, s.handlePublicMqttClientDelete)

	route("GET", "/backup", admin, s.handlePublicBackup)
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

func (s *ApiModule) withApiKey(scope models.ApiKeyScope, h publicHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			w.Header().Set("WWW-Authenticate", `Bearer realm="ubergang"`)
			writePublicError(w, http.StatusUnauthorized, "unauthorized",
				"An API key is required, as \"Authorization: Bearer <key>\"")
			return
		}
		key, err := s.auth.AuthenticateApiKey(token)
		if err != nil {
			s.log.Warnf("Rejected public API request to %s %s: %v", r.Method, r.URL.Path, err)
			w.Header().Set("WWW-Authenticate", `Bearer realm="ubergang", error="invalid_token"`)
			writePublicError(w, http.StatusUnauthorized, "unauthorized", "Invalid API key")
			return
		}
		if !auth.ApiKeyHasScope(key, scope) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="ubergang", error="insufficient_scope"`)
			writePublicError(w, http.StatusForbidden, "insufficient_scope",
				fmt.Sprintf("This API key doesn't have the %q scope", apiKeyScopeName(scope)))
			return
		}
		h(w, r, key)
	}
}

func writePublicJson(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	// Responses may contain passwords.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(body)
}

func writePublicError(w http.ResponseWriter, status int, code, message string) {
	writePublicJson(w, status, api.PublicError{
		Error: api.PublicErrorDetails{Code: code, Message: message},
	})
}

// parsePublicJson decodes the request body into `req`, rejecting unknown
// fields so that typos aren't silently ignored. It writes an error response
// and returns false on failure.
func parsePublicJson(w http.ResponseWriter, r *http.Request, req interface{}) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writePublicError(w, http.StatusUnsupportedMediaType, "unsupported_media_type",
			"The request body must be JSON, with \"Content-Type: application/json\"")
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxPublicBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(req); err != nil {
		writePublicError(w, http.StatusBadRequest, publicErrBadRequest, "Invalid JSON body: "+err.Error())
		return false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		writePublicError(w, http.StatusBadRequest, publicErrBadRequest, "Invalid JSON body: unexpected data after the JSON value")
		return false
	}
	return true
}

// validatePublicId checks an ID that will be used in URLs.
func validatePublicId(field, id string) error {
	switch {
	case id == "":
		return fmt.Errorf("%s is required", field)
	case len(id) > maxPublicIdLength:
		return fmt.Errorf("%s must be at most %d bytes", field, maxPublicIdLength)
	case !utf8.ValidString(id):
		return fmt.Errorf("%s must be valid UTF-8", field)
	case strings.Contains(id, "/"):
		return fmt.Errorf("%s must not contain '/'", field)
	case strings.TrimSpace(id) != id:
		return fmt.Errorf("%s must not start or end with whitespace", field)
	case strings.IndexFunc(id, unicode.IsControl) >= 0:
		return fmt.Errorf("%s must not contain control characters", field)
	}
	return nil
}
