package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	// Static assets

	"boivie/ubergang/server/auth"
	"boivie/ubergang/server/backends"
	"boivie/ubergang/server/db"
	uglog "boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/mqtt"
	"boivie/ubergang/server/proxy"
	"boivie/ubergang/server/rest"
	"boivie/ubergang/server/session"
	"boivie/ubergang/server/ssh_server"
	"boivie/ubergang/server/tls"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"

	"github.com/gorilla/mux"
)

var flgSshPort = flag.Int("ssh", 10022, "Port for SSH server")
var flgHttpsPort = flag.Int("https", 10443, "Port for HTTPS server")
var flgHttpPort = flag.Int("http", 10080, "Port for HTTP server")
var flgProxyTestPort = flag.Int("proxy-test", 0, "Port for proxy test")
var flgMetricsPort = flag.Int("metrics", 9090, "Port for metrics server")
var flgMqttPort = flag.Int("mqtt", 1883, "Port for MQTT proxy server")
var flgMqttTlsPort = flag.Int("mqtt-tls", 8883, "Port for MQTT TLS proxy server")
var flgMqttServer = flag.String("mqtt-server", "", "MQTT server")
var flgLocalDev = flag.Bool("local-dev", false, "Local development")
var flgVerbose = flag.Bool("verbose", false, "Verbose logs")

var (
	httpRequestsTotalMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ubergang_http_requests_total",
		Help: "The total number of HTTP requests",
	}, []string{"host", "status"})
	httpRequestsSizeMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ubergang_http_request_size_bytes",
		Help: "The total size of incoming requests"}, []string{"host"})
	httpResponseSizeMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ubergang_http_response_size_bytes",
		Help: "The total size of HTTP responses"}, []string{"host"})
	httpRequestLatencyMetric = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "ubergang_http_request_duration_seconds",
		Help: "HTTP request latency",
	}, []string{"host"})
)

type Server struct {
	config         *models.Configuration
	db             *db.DB
	tlsManager     tls.TlsManager
	assets         *embed.FS
	updateAccessed chan *models.Session
	backendManager *backends.BackendManager
	log            *uglog.Log
	session        *session.SessionStore
	auth           *auth.Auth
	api            *rest.ApiModule
	proxy          *proxy.Proxy
	sshServer      *ssh_server.SSHServer
	mqttProxy      *mqtt.MqttProxy
	mqttPublisher  mqtt.MQTTPublisher
}

func NewServer(dbFile string, assets *embed.FS) *Server {
	log := uglog.NewLogger(uglog.Fields{})
	log.Debug("Debug logging enabled")
	log.Infof("Using database at %s", dbFile)
	db, err := db.New(log, dbFile)
	if err != nil {
		log.Fatalf("Failed to load database: %v", err)
		os.Exit(1)
	}

	config, err := db.GetConfiguration()
	if err != nil {
		config = &models.Configuration{}
	}
	if *flgVerbose {
		uglog.SetLogLevel(logrus.DebugLevel)
	}

	var tlsManager tls.TlsManager
	backends := backends.New(db, log)

	// Check if server is configured
	isConfigured := config.Email != "" && config.SiteFqdn != "" && config.AdminFqdn != ""

	if !isConfigured {
		// BOOTSTRAP MODE: Use self-signed certificate
		log.Infof("Server not configured - entering bootstrap mode with self-signed certificate")

		// Try to load existing bootstrap certificate from database
		certPEM, keyPEM, err := db.GetSelfSignedCert()
		if err != nil {
			// Generate new bootstrap certificate
			log.Infof("Generating new bootstrap certificate (30-year validity)")
			localManager := tls.NewLocalCertTlsManager()
			certPEM, keyPEM = localManager.ExportPEM()

			// Store in database for persistence across restarts
			if err := db.UpdateSelfSignedCert(certPEM, keyPEM); err != nil {
				log.Warnf("Failed to store bootstrap certificate: %v", err)
			} else {
				log.Infof("Bootstrap certificate stored in database")
			}
			tlsManager = localManager
		} else {
			// Use existing bootstrap certificate from database
			log.Infof("Using existing bootstrap certificate from database")
			tlsManager, err = tls.NewLocalCertTlsManagerFromBytes(certPEM, keyPEM)
			if err != nil {
				log.Fatalf("Failed to load bootstrap certificate: %v", err)
			}
		}
	} else if config.IsInTestMode {
		// TEST MODE: Use self-signed certs
		log.Infof("Using self-signed certs (test mode)")
		tlsManager = tls.NewLocalCertTlsManager()
	} else {
		// PRODUCTION MODE: Use Let's Encrypt
		// Check if Google Cloud DNS credentials are configured via environment variable
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
			log.Infof("Using LetsEncrypt certificates via certmagic with DNS-01 (wildcard for *.%s)", config.SiteFqdn)
		} else {
			log.Infof("Using LetsEncrypt certificates via certmagic with HTTP-01")
		}

		tlsManager = tls.NewCertMagicTlsManager(db, config.Email, config.SiteFqdn, func(ctx context.Context, host string) error {
			// Allow admin FQDN
			if host == config.AdminFqdn {
				return nil
			}

			// Allow registered backends
			if _, err := backends.Lookup(host); err == nil {
				return nil
			}

			// When using wildcard certificates, allow any subdomain of SiteFqdn
			if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" && config.SiteFqdn != "" {
				// Check if host is a subdomain of SiteFqdn or matches exactly
				if host == config.SiteFqdn || strings.HasSuffix(host, "."+config.SiteFqdn) {
					return nil
				}
			}

			return fmt.Errorf("the hostname %s can't be found in the registered list of backends", host)
		})
	}

	session := session.NewSessionStore(log, db)
	auth := auth.New(log, db)
	updateAccessed := make(chan *models.Session)
	mqttProxy := mqtt.New(log, config, db, tlsManager, *flgMqttServer)

	// Create MQTT publisher if broker is configured
	var mqttPublisher mqtt.MQTTPublisher = nil
	if *flgMqttServer != "" {
		mqttPublisher = mqtt.NewPublisher(log, *flgMqttServer)
		log.Infof("Created MQTT publisher for broker %s", *flgMqttServer)
	}

	s := &Server{
		config:         config,
		db:             db,
		tlsManager:     tlsManager,
		assets:         assets,
		updateAccessed: updateAccessed,
		log:            log,
		backendManager: backends,
		session:        session,
		auth:           auth,
		api:            rest.New(config, db, log, session, auth, mqttProxy),
		proxy:          proxy.New(config, log, session, updateAccessed, backends, mqttPublisher),
		sshServer:      ssh_server.New(log, config, db, backends),
		mqttProxy:      mqttProxy,
		mqttPublisher:  mqttPublisher,
	}

	go s.sessionAccessUpdater()
	//go db.PerformPeriodicBackups()
	return s
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := negroni.NewResponseWriter(w)
			next.ServeHTTP(lrw, r)
			duration := time.Since(start)
			host := strings.ToLower(r.Host)
			logger.Printf("%s %s%s %s\n", r.Method, host, r.URL.Path, r.RemoteAddr)
			httpRequestsSizeMetric.WithLabelValues(host).Add(float64(r.ContentLength))
			httpResponseSizeMetric.WithLabelValues(host).Add(float64(lrw.Size()))
			httpRequestsTotalMetric.WithLabelValues(host, strconv.Itoa(lrw.Status())).Inc()
			httpRequestLatencyMetric.WithLabelValues(host).Observe(duration.Seconds())
		})
	}
}

func makeServerFromMux(logger *log.Logger, mux *mux.Router) *http.Server {
	return &http.Server{
		IdleTimeout: 120 * time.Second,
		Handler:     logging(logger)(mux),
	}
}

func (s *Server) Serve() error {
	defer s.db.Close()
	defer func() {
		if s.mqttPublisher != nil {
			s.log.Info("Shutting down MQTT publisher...")
			if err := s.mqttPublisher.Close(); err != nil {
				s.log.Warnf("Error closing MQTT publisher: %v", err)
			}
		}
	}()

	sshKeyPem, err := s.db.GetSshServerKey()
	if err != nil {
		fmt.Println("Generating new SSH key")
		sshKeyPem, err = auth.GenerateSshKey()
		if err != nil {
			log.Fatalf("Unable to generate SSH key: %v", err)
		}
		err = s.db.UpdateSshServerKey(sshKeyPem)
		if err != nil {
			log.Fatalf("Unable to save SSH key: %v", err)
		}
	}

	go s.sshServer.ServeSSH(sshKeyPem, *flgSshPort)
	go s.ServeMetrics()
	go s.httpsServer()

	if *flgMqttServer != "" {
		s.mqttProxy.Start(*flgMqttPort, *flgMqttTlsPort)
	}
	if *flgProxyTestPort != 0 {
		go s.proxyTestServer()
	}
	s.httpServer()
	return nil
}

func (s *Server) ServeMetrics() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	r.Handle("/debug/pprof/block", pprof.Handler("block"))
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/backup", s.db.BackupHttpHandler())
	metricSrv := makeServerFromMux(logger, r)
	metricSrv.Addr = fmt.Sprintf(":%d", *flgMetricsPort)

	fmt.Printf("Starting Metrics server on %s\n", metricSrv.Addr)
	err := metricSrv.ListenAndServe()
	if err != nil {
		log.Fatalf("metricSrv.ListenAndServe() failed with %s", err)
	}
}

func (s *Server) httpServer() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		newURI := "https://" + r.Host + r.URL.String()
		http.Redirect(w, r, newURI, http.StatusFound)
	})
	httpSrv := makeServerFromMux(logger, r)
	httpSrv.Handler = s.tlsManager.HTTPHandler(httpSrv.Handler)
	httpSrv.Addr = fmt.Sprintf(":%d", *flgHttpPort)
	fmt.Printf("Starting HTTP server on %s\n", httpSrv.Addr)
	err := httpSrv.ListenAndServe()
	if err != nil {
		log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}
}

func (s *Server) proxyTestServer() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Register proxy test endpoints (must be before proxy catch-all)
	r := mux.NewRouter()
	s.api.RegisterTestEndpoints(r)
	httpSrv := makeServerFromMux(logger, r)

	httpSrv.Handler = s.tlsManager.HTTPHandler(httpSrv.Handler)
	httpSrv.Addr = fmt.Sprintf(":%d", *flgProxyTestPort)
	fmt.Printf("Starting proxy test server on %s\n", httpSrv.Addr)
	err := httpSrv.ListenAndServe()
	if err != nil {
		log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}
}

// fsFunc is short-hand for constructing a http.FileSystem
// implementation
type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

// AssetHandler returns an http.Handler that will serve files from
// the Assets embed.FS.  When locating a file, it will strip the given
// prefix from the request and prepend the root to the filesystem
// lookup: typical prefix might be /web/, and root would be build.
func AssetHandler(assets *embed.FS, root string) http.Handler {
	handler := fsFunc(func(name string) (fs.File, error) {
		assetPath := path.Join(root, name)

		log.Printf("Getting asset: %s -> %s\n", name, assetPath)

		// If we can't find the asset, return the default index.html
		// content
		f, err := assets.Open(assetPath)
		if os.IsNotExist(err) {
			return assets.Open("web/dist/index.html")
		}

		// Otherwise assume this is a legitimate request routed
		// correctly
		return f, err
	})

	return http.FileServer(http.FS(handler))
}

type localFrontend struct{}

func (b *localFrontend) Type() string              { return "dev-frontend" }
func (b *localFrontend) Host() string              { return "localhost" }
func (b *localFrontend) Headers() []*models.Header { return []*models.Header{} }
func (b *localFrontend) NeedsAuth() bool           { return false }
func (b *localFrontend) URL() *url.URL {
	u, _ := url.Parse("http://localhost:5173")
	return u
}
func (b *localFrontend) JsScript() *goja.Program {
	return nil
}
func (b *localFrontend) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{
		Timeout:   2 * time.Second,
		KeepAlive: 5 * time.Second,
		DualStack: true,
	}).DialContext(ctx, network, address)
}

func (s *Server) httpsServer() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	r := mux.NewRouter()

	// Check if server is configured
	isConfigured := s.config.Email != "" && s.config.SiteFqdn != "" && s.config.AdminFqdn != ""

	if !s.config.IsInTestMode && !isConfigured {
		// BOOTSTRAP MODE: Ignore Host header, serve bootstrap UI on all hosts
		logger.Printf("Bootstrap mode: serving setup UI on all hosts")

		// Register bootstrap API endpoints (without Host requirement)
		s.api.RegisterBootstrapEndpoints(r)

		// Serve bootstrap frontend
		if *flgLocalDev {
			r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.proxy.ProxyRequest(w, r, &localFrontend{}, nil, nil)
			})
		} else {
			r.PathPrefix("/").Handler(AssetHandler(s.assets, "web/dist"))
		}
	} else {
		// NORMAL MODE: Use host-based routing
		logger.Printf("Registering API endpoint at %s", s.config.AdminFqdn)
		s.api.RegisterEndpoints(r)
		r.Host(s.config.AdminFqdn).Path("/authorize").HandlerFunc(s.proxy.HandleAuthorize)

		if *flgLocalDev {
			r.Host(s.config.AdminFqdn).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.proxy.ProxyRequest(w, r, &localFrontend{}, nil, nil)
			})
		} else {
			r.Host(s.config.AdminFqdn).Handler(AssetHandler(s.assets, "web/dist"))
		}
		r.PathPrefix("/").HandlerFunc(s.proxy.ProxyHandler)
	}

	httpsSrv := makeServerFromMux(logger, r)
	httpsSrv.Addr = fmt.Sprintf(":%d", *flgHttpsPort)
	httpsSrv.TLSConfig = s.tlsManager.TLSConfig()
	fmt.Printf("Starting HTTPS server on %s\n", httpsSrv.Addr)
	err := httpsSrv.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalf("httpsSrv.ListendAndServeTLS() failed with %s", err)
	}
}
