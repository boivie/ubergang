package proxy

import (
	"boivie/ubergang/server/backends"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/mqtt"
	"boivie/ubergang/server/scripting"
	"boivie/ubergang/server/session"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var backendConnectionErrorsTotalMetric = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "ubergang_backend_connection_errors_total",
	Help: "The total number of HTTP requests",
}, []string{"host", "backend"})

type Proxy struct {
	config         *models.Configuration
	backends       *backends.BackendManager
	log            *log.Log
	session        *session.SessionStore
	updateAccessed chan *models.Session
	mqttPublisher  mqtt.MQTTPublisher
}

func New(
	config *models.Configuration,
	log *log.Log,
	session *session.SessionStore,
	updateAccessed chan *models.Session,
	backends *backends.BackendManager,
	mqttPublisher mqtt.MQTTPublisher) *Proxy {
	return &Proxy{config, backends, log, session, updateAccessed, mqttPublisher}
}

func (s *Proxy) redirectAuthorizeInvalidSession(w http.ResponseWriter, r *http.Request) {
	redirect := r.URL.RequestURI()
	url := fmt.Sprintf("https://%s/authorize?rd=https://%s%s",
		s.config.AdminFqdn, r.Host, redirect)
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Proxy) serveHandleTrampoline(w http.ResponseWriter, r *http.Request) bool {
	value := r.URL.Query().Get("_ubergang_session")
	if value == "" {
		return false
	}
	_, session, err := s.session.DecodeSessionCookie(value, true)
	if err != nil {
		s.log.Warnf("Failed to find session from trampoline: %v", err)
		s.redirectsigninInvalidSession(w, r)
		return false
	}

	u := r.URL
	q := u.Query()
	q.Del("_ubergang_session")
	u.RawQuery = q.Encode()
	http.SetCookie(w, s.session.CreateSessionCookie(session))
	http.Redirect(w, r, u.String(), http.StatusFound)
	s.log.Info("Set session cookie")
	return true
}

func isAllowed(user *models.User, host string) bool {
	if user.IsAdmin {
		return true
	}
	for _, allowed := range user.AllowedHosts {
		if allowed == host {
			return true
		}
	}
	return false
}

func (s *Proxy) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	if s.serveHandleTrampoline(w, r) {
		return
	}

	backend, err := s.backends.Lookup(r.Host)
	if err != nil {
		s.log.Warnf("Failed to find backend %s: %v", r.Host, err)
		http.Error(w, "No backend found", http.StatusBadGateway)
		return
	}
	s.log.Debugf("Resolved %s to %s backend (%s)", r.Host, backend.Type(), backend.URL())

	var user *models.User = nil
	var session *models.Session = nil

	if backend.NeedsAuth() {
		user, session, err = s.session.Get(r)
		if err != nil {
			s.redirectAuthorizeInvalidSession(w, r)
			return
		}
		if !isAllowed(user, backend.Host()) {
			s.log.Warnf("User %s is not allowed to access %s", user.Email, backend.Host())
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	if backend.JsScript() != nil {
		js, err := scripting.NewJSProxy(backend.JsScript(), s.mqttPublisher)
		if err != nil {
			s.log.Warnf("Failed to create JS proxy: %v", err)
			http.Error(w, "Failed to create JS proxy", http.StatusInternalServerError)
			return
		}

		if !js.MatchAndExecute(w, r) {
			return
		}
	}

	s.ProxyRequest(w, r, backend, user, session)
}

func evaluate(value string, variables map[string]string) string {
	if strings.HasPrefix(value, "$") {
		if v, ok := variables[value]; ok {
			return v
		}
		return ""
	}
	return value
}

func createTLSConfig(pinnedFingerprint string) (*tls.Config, error) {
	if pinnedFingerprint == "" {
		// No pinned certificate - use system certificate verification
		return &tls.Config{
			InsecureSkipVerify: false,
		}, nil
	}

	// Certificate pinning is enabled - verify the fingerprint
	return &tls.Config{
		// codeql[go/disabled-certificate-check]
		InsecureSkipVerify: true, // We'll do custom verification
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			// Check if any certificate in the chain matches the pinned fingerprint
			for _, rawCert := range rawCerts {
				hash := sha256.Sum256(rawCert)
				fingerprint := hex.EncodeToString(hash[:])

				if strings.EqualFold(fingerprint, pinnedFingerprint) {
					return nil
				}
			}

			return fmt.Errorf("certificate fingerprint mismatch: none of the certificates in the chain match the pinned fingerprint")
		},
	}, nil
}

func (s *Proxy) ProxyRequest(w http.ResponseWriter, r *http.Request, backend backends.Backend, user *models.User, session *models.Session) {
	upstream := backend.URL()

	director := func(req *http.Request) {
		variables := map[string]string{
			"$http_host":     req.Host,
			"$upstream_host": upstream.Host,
		}
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", "https")
		if user != nil {
			req.Header.Set("X-Forwarded-Email", user.Email)
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}

		req.URL.Scheme = upstream.Scheme
		req.URL.Host = upstream.Host
		req.Host = backend.Host()

		for _, header := range backend.Headers() {
			if strings.ToLower(header.Name) == "host" {
				req.Host = evaluate(header.Value, variables)
			} else {
				if header.Value == "" {
					delete(req.Header, header.Name)
				} else {
					req.Header.Set(header.Name, evaluate(header.Value, variables))
				}
			}
		}
	}

	tlsConfig, err := createTLSConfig(backend.PinnedCertificateFingerprint())
	if err != nil {
		s.log.Warnf("Failed to create TLS config: %v", err)
		http.Error(w, "Failed to create TLS config", http.StatusInternalServerError)
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: director,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           backend.DialContext,
			TLSClientConfig:       tlsConfig,
			MaxConnsPerHost:       4,
			MaxIdleConns:          20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableCompression:    false,
			DisableKeepAlives:     true,
		},
		FlushInterval: 100 * time.Millisecond,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			s.backendConnectionError(backend, w, r, err)
		},
		ModifyResponse: nil,
	}
	proxy.ServeHTTP(w, r)
}

func (s *Proxy) backendConnectionError(backend backends.Backend, w http.ResponseWriter, r *http.Request, err error) {
	backendConnectionErrorsTotalMetric.WithLabelValues(strings.ToLower(r.Host), backend.URL().Host).Inc()
	s.log.Warnf("Failed to connect to backend %v: %v", backend.URL(), err)

	// Check if this is a TLS certificate validation error
	errorMsg := "Failed to connect to backend"
	statusCode := http.StatusBadGateway

	// Check for certificate-related errors
	if err != nil {
		errStr := err.Error()
		// Check for our custom certificate pinning error
		if strings.Contains(errStr, "certificate fingerprint mismatch") {
			errorMsg = "Certificate validation failed: The backend's SSL certificate does not match the pinned fingerprint"
			statusCode = http.StatusBadGateway
		} else if strings.Contains(errStr, "x509:") || strings.Contains(errStr, "certificate") || strings.Contains(errStr, "tls:") {
			// Check for standard x509/TLS certificate errors
			switch {
			case strings.Contains(errStr, "unknown authority") || strings.Contains(errStr, "certificate signed by unknown authority"):
				errorMsg = "Certificate validation failed: The backend's SSL certificate is not trusted by the system"
			case strings.Contains(errStr, "certificate has expired") || strings.Contains(errStr, "certificate is not yet valid"):
				errorMsg = "Certificate validation failed: The backend's SSL certificate has expired or is not yet valid"
			case strings.Contains(errStr, "certificate is valid for"):
				errorMsg = "Certificate validation failed: The backend's SSL certificate hostname does not match"
			default:
				errorMsg = fmt.Sprintf("Certificate validation failed: %v", err)
			}
			statusCode = http.StatusBadGateway
		}
	}

	http.Error(w, errorMsg, statusCode)
}
