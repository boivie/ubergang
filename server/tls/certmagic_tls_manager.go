package tls

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	cert_pkg "boivie/ubergang/server/cert"
	"boivie/ubergang/server/db"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/googleclouddns"
)

type CertMagicTlsManager struct {
	config         *certmagic.Config
	issuer         *certmagic.ACMEIssuer
	usingDNS01     bool
	wildcardDomain string
}

// HostPolicyFunc is a function that validates whether a hostname should get a certificate
type HostPolicyFunc func(ctx context.Context, host string) error

// NewCertMagicTlsManager creates a new TLS manager using certmagic
// DNS-01 challenges are used if GOOGLE_APPLICATION_CREDENTIALS env var is set
// Otherwise, HTTP-01 challenges will be used
// wildcardDomain should be the domain for the wildcard cert (e.g., "example.com" for "*.example.com")
func NewCertMagicTlsManager(db *db.DB, email string, wildcardDomain string, hostPolicy HostPolicyFunc) *CertMagicTlsManager {
	// Create custom storage backed by the database
	storage := cert_pkg.NewCertMagicStorage(db)

	// Check if Google Cloud credentials are available via standard env var
	gcpCredentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	// Configure ACME issuer based on whether we have GCP credentials
	issuerConfig := certmagic.ACMEIssuer{
		CA:                      certmagic.LetsEncryptProductionCA,
		Agreed:                  true,
		Email:                   email,
		DisableTLSALPNChallenge: true,
	}

	// If GCP credentials are provided, use DNS-01 for wildcard certificates
	if gcpCredentialsPath != "" {
		// Load GCP credentials to extract project ID
		gcpCredsJSON, err := os.ReadFile(gcpCredentialsPath)
		if err != nil {
			fmt.Printf("ERROR: Failed to read GCP credentials from %s: %v\n", gcpCredentialsPath, err)
			fmt.Println("Falling back to HTTP-01 challenge mode")
			gcpCredentialsPath = "" // Disable DNS-01
		} else {
			// Extract project_id from the service account JSON
			var gcpCreds struct {
				ProjectID string `json:"project_id"`
			}
			if err := json.Unmarshal(gcpCredsJSON, &gcpCreds); err != nil {
				fmt.Printf("ERROR: Failed to parse GCP credentials: %v\n", err)
				fmt.Println("Falling back to HTTP-01 challenge mode")
				gcpCredentialsPath = "" // Disable DNS-01
			} else if gcpCreds.ProjectID == "" {
				fmt.Println("ERROR: GCP credentials missing project_id field")
				fmt.Println("Falling back to HTTP-01 challenge mode")
				gcpCredentialsPath = "" // Disable DNS-01
			} else {
				// Configure Google Cloud DNS provider
				// The provider will automatically use GOOGLE_APPLICATION_CREDENTIALS env var
				dnsProvider := &googleclouddns.Provider{
					Project: gcpCreds.ProjectID,
				}

				// Enable DNS-01 challenge with DNS propagation settings
				issuerConfig.DNS01Solver = &certmagic.DNS01Solver{
					DNSManager: certmagic.DNSManager{
						DNSProvider:        dnsProvider,
						PropagationTimeout: 2 * time.Minute,  // Max wait for DNS propagation
						PropagationDelay:   10 * time.Second, // Initial wait before checking
					},
				}
				issuerConfig.DisableHTTPChallenge = true
			}
		}
	}

	// If not using DNS-01, configure HTTP-01
	if gcpCredentialsPath == "" {
		// Use HTTP-01 challenge with distributed solving
		issuerConfig.DisableHTTPChallenge = false
		issuerConfig.DisableDistributedSolvers = false
		issuerConfig.ListenHost = "0.0.0.0"
		issuerConfig.AltHTTPPort = 0
	}

	// Create ACME issuer for Let's Encrypt
	issuer := certmagic.NewACMEIssuer(&certmagic.Config{
		Storage: storage,
	}, issuerConfig)

	// Create certmagic config with a cache
	// We need to declare config first so the closure can reference it
	var config *certmagic.Config

	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(cert certmagic.Certificate) (*certmagic.Config, error) {
			// Return the config we're building, not a new default
			// This ensures the same storage and issuer configuration is used
			return config, nil
		},
	})

	config = certmagic.New(cache, certmagic.Config{
		Storage:            storage,
		Issuers:            []certmagic.Issuer{issuer},
		RenewalWindowRatio: 0.33, // Start renewal when 1/3 of lifetime remains (~30 days for 90-day certs)
		OnDemand: &certmagic.OnDemandConfig{
			DecisionFunc: hostPolicy,
		},
	})

	usingDNS01 := gcpCredentialsPath != ""

	// If using DNS-01, pre-obtain the wildcard certificate in the background
	if usingDNS01 && wildcardDomain != "" {
		wildcardName := "*." + wildcardDomain
		fmt.Printf("Scheduling wildcard certificate acquisition for: %s\n", wildcardName)
		fmt.Println("This will happen in the background. Server will start immediately.")

		// Obtain the wildcard certificate in the background with timeout
		// Server starts immediately, wildcard cert is obtained async
		go func() {
			// Use a context with timeout to prevent indefinite hangs
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			fmt.Printf("Obtaining wildcard certificate for: %s\n", wildcardName)
			fmt.Println("This may take 1-2 minutes for DNS propagation...")

			err := config.ObtainCertSync(ctx, wildcardName)
			if err != nil {
				fmt.Printf("WARNING: Failed to obtain wildcard certificate for %s: %v\n", wildcardName, err)
				fmt.Println("Server will use on-demand certificate issuance as fallback")
			} else {
				fmt.Printf("SUCCESS: Wildcard certificate obtained for %s\n", wildcardName)
			}
		}()
	}

	return &CertMagicTlsManager{
		config:         config,
		issuer:         issuer,
		usingDNS01:     usingDNS01,
		wildcardDomain: wildcardDomain,
	}
}

// TLSConfig returns a TLS configuration for the HTTPS server
func (m *CertMagicTlsManager) TLSConfig() *tls.Config {
	return m.config.TLSConfig()
}

// HTTPHandler wraps an HTTP handler to handle ACME HTTP-01 challenges
// In Phase 1, this handles certificate validation challenges using distributed solving
// In Phase 2 (DNS-01), this will only handle HTTP->HTTPS redirects
func (m *CertMagicTlsManager) HTTPHandler(fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this looks like an ACME HTTP challenge request
		if certmagic.LooksLikeHTTPChallenge(r) {
			// Use certmagic's distributed solver to handle the challenge
			// This loads challenge info from storage and serves the key authorization
			ctx := r.Context()

			// Load challenge tokens for this domain from storage
			// The CA URL is normalized to a directory name by certmagic
			// For Let's Encrypt production: acme-v02.api.letsencrypt.org-directory
			challengeInfoPath := "acme/acme-v02.api.letsencrypt.org-directory/challenge_tokens/" + r.Host + ".json"

			data, err := m.config.Storage.Load(ctx, challengeInfoPath)
			if err == nil && len(data) > 0 {
				// Parse the JSON to get challenge data
				// The JSON structure is: {"type":"http-01", "token":"...", "keyAuthorization":"..."}
				var challenge struct {
					Type             string `json:"type"`
					Token            string `json:"token"`
					KeyAuthorization string `json:"keyAuthorization"`
				}

				if err := json.Unmarshal(data, &challenge); err == nil && challenge.KeyAuthorization != "" {
					// Serve the key authorization to the ACME server
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(challenge.KeyAuthorization))
					return
				}
			}
		}

		// Not a challenge request or challenge not found, pass through to fallback
		fallback.ServeHTTP(w, r)
	})
}

// GetCertificate dynamically obtains a certificate for the given client hello
func (m *CertMagicTlsManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// Use certmagic's GetCertificate which handles on-demand cert issuance
	return m.config.GetCertificate(hello)
}
