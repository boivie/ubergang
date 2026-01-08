package tls

import (
	"crypto/tls"
	"net/http"
)

type TlsManager interface {
	HTTPHandler(fallback http.Handler) http.Handler
	TLSConfig() *tls.Config
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}
