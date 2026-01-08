package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"time"
)

type LocalCertTlsManager struct {
	cert       tls.Certificate
	privateKey *ecdsa.PrivateKey
}

func NewLocalCertTlsManager() *LocalCertTlsManager {
	// Generate a new private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	// Create a new template for the certificate
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Ubergang",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 10950), // 30 years

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create a new certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		panic(err)
	}

	// Create a new TLS certificate
	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privateKey,
	}

	return &LocalCertTlsManager{cert, privateKey}
}

// NewLocalCertTlsManagerFromBytes creates a LocalCertTlsManager from PEM-encoded certificate and key
func NewLocalCertTlsManagerFromBytes(certPEM, keyPEM []byte) (*LocalCertTlsManager, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	// Extract the private key
	privateKey, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, err
	}

	return &LocalCertTlsManager{cert, privateKey}, nil
}

func (m *LocalCertTlsManager) TLSConfig() *tls.Config {

	// Return a new TLS config
	return &tls.Config{
		Certificates: []tls.Certificate{m.cert},
	}
}

func (m *LocalCertTlsManager) HTTPHandler(fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallback.ServeHTTP(w, r)
	})
}

func (m *LocalCertTlsManager) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return &m.cert, nil
}

// ExportPEM returns the certificate and private key as PEM-encoded bytes
func (m *LocalCertTlsManager) ExportPEM() (certPEM []byte, keyPEM []byte) {
	// Encode certificate
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: m.cert.Certificate[0],
	})

	// Encode private key
	keyBytes, err := x509.MarshalECPrivateKey(m.privateKey)
	if err != nil {
		panic(err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	return certPEM, keyPEM
}
