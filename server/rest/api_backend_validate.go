package rest

import (
	"boivie/ubergang/server/api"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net"
	"net/http"
	"net/url"
	"time"
)

func formatCertificate(cert *x509.Certificate) api.ApiCertificate {
	// Calculate SHA256 fingerprint
	hash := sha256.Sum256(cert.Raw)
	fingerprint := hex.EncodeToString(hash[:])

	return api.ApiCertificate{
		Subject:            cert.Subject.String(),
		Issuer:             cert.Issuer.String(),
		NotBefore:          cert.NotBefore.Format(time.RFC3339),
		NotAfter:           cert.NotAfter.Format(time.RFC3339),
		SerialNumber:       cert.SerialNumber.String(),
		DNSNames:           cert.DNSNames,
		SignatureAlgorithm: cert.SignatureAlgorithm.String(),
		SHA256Fingerprint:  fingerprint,
	}
}

func (s *ApiModule) handleBackendValidate(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}
	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}
	upstreamURLQuery := r.URL.Query().Get("upstream_url")
	upstreamURL, err := url.Parse(upstreamURLQuery)
	if err != nil {
		jsonify(w, api.ApiValidateBackendResponse{
			Reachable: false,
			Error:     "Invalid upstream URL: " + err.Error(),
		})
		return
	}

	// Determine the target address for connection
	targetHost := upstreamURL.Host
	if upstreamURL.Port() == "" {
		if upstreamURL.Scheme == "https" {
			targetHost = upstreamURL.Host + ":443"
		} else {
			targetHost = upstreamURL.Host + ":80"
		}
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt to dial the backend
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", targetHost)
	if err != nil {

		jsonify(w, api.ApiValidateBackendResponse{
			Reachable: false,
			Error:     "Connection failed: " + err.Error(),
		})
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	// If HTTPS, upgrade to TLS and capture certificate chain
	if upstreamURL.Scheme == "https" {
		// First try with valid certificate check
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         upstreamURL.Hostname(),
		}

		tlsConn := tls.Client(conn, tlsConfig)
		err = tlsConn.HandshakeContext(ctx)

		validCertificate := false
		var peerCertificates []*x509.Certificate

		if err == nil {
			validCertificate = true
			state := tlsConn.ConnectionState()
			peerCertificates = state.PeerCertificates
		} else {
			// Failed to validate certificate (or other error)
			// Try again without verification
			_ = conn.Close()

			conn, err = dialer.DialContext(ctx, "tcp", targetHost)
			if err != nil {
				jsonify(w, api.ApiValidateBackendResponse{
					Reachable: false,
					Error:     "Connection failed on retry: " + err.Error(),
				})
				return
			}
			defer func() {
				_ = conn.Close()
			}()

			// codeql[go/disabled-certificate-check]
			tlsConfig.InsecureSkipVerify = true
			tlsConn = tls.Client(conn, tlsConfig)
			err = tlsConn.HandshakeContext(ctx)
			if err != nil {
				jsonify(w, api.ApiValidateBackendResponse{
					Reachable: true,
					TLS:       true,
					Error:     "TLS handshake failed: " + err.Error(),
				})
				return
			}
			state := tlsConn.ConnectionState()
			peerCertificates = state.PeerCertificates
		}

		certificates := make([]api.ApiCertificate, 0, len(peerCertificates))
		for _, cert := range peerCertificates {
			certificates = append(certificates, formatCertificate(cert))
		}

		jsonify(w, api.ApiValidateBackendResponse{
			Reachable:        true,
			TLS:              true,
			ValidCertificate: validCertificate,
			Certificates:     certificates,
		})
		return
	}

	// Non-TLS connection successful
	jsonify(w, api.ApiValidateBackendResponse{
		Reachable: true,
		TLS:       false,
	})
}
