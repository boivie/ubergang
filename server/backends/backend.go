package backends

import (
	"boivie/ubergang/server/access"
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"context"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"google.golang.org/protobuf/proto"
)

type Backend interface {
	Type() string
	Host() string
	Headers() []*models.Header
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	AccessPolicy() *access.Policy
	URL() *url.URL
	JsScript() *goja.Program
	PinnedCertificateFingerprint() string
}

type BackendManager struct {
	db        *db.DB
	log       *log.Log
	ephemeral map[string]Backend

	// Compiled backends, to avoid compiling scripts and rules on every request.
	mu       sync.Mutex
	compiled map[string]*localBackend
}

func New(db *db.DB, log *log.Log) *BackendManager {
	return &BackendManager{db: db, log: log, ephemeral: make(map[string]Backend), compiled: make(map[string]*localBackend)}
}

type localBackend struct {
	log     *log.Log
	host    string
	backend *models.Backend
	url     *url.URL
	program *goja.Program
	policy  *access.Policy
}

func (b *localBackend) Type() string                 { return "local" }
func (b *localBackend) Host() string                 { return b.host }
func (b *localBackend) Headers() []*models.Header    { return b.backend.Headers }
func (b *localBackend) AccessPolicy() *access.Policy { return b.policy }
func (b *localBackend) URL() *url.URL                { return b.url }
func (b *localBackend) JsScript() *goja.Program      { return b.program }
func (b *localBackend) PinnedCertificateFingerprint() string {
	return b.backend.PinnedCertificateFingerprint
}

func (b *localBackend) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	newAddress := b.url.Host
	if b.url.Port() == "" {
		newAddress = b.url.Host + ":80"
	}

	b.log.Infof("Dialing %s %s / %s", network, address, newAddress)
	return (&net.Dialer{
		Timeout:   2 * time.Second,
		KeepAlive: 5 * time.Second,
	}).DialContext(ctx, network, newAddress)
}

func (m *BackendManager) Lookup(host string) (Backend, error) {
	if strings.Contains(host, ":") {
		host = host[0:strings.Index(host, ":")]
	}

	backend, err := m.db.GetBackend(host)
	if err != nil {
		m.mu.Lock()
		delete(m.compiled, host)
		m.mu.Unlock()
		if b, ok := m.ephemeral[host]; ok {
			return b, nil
		}
		return nil, err
	}

	m.mu.Lock()
	cached, ok := m.compiled[host]
	m.mu.Unlock()
	if ok && proto.Equal(cached.backend, backend) {
		return cached, nil
	}

	compiled, err := m.compile(host, backend)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.compiled[host] = compiled
	m.mu.Unlock()
	return compiled, nil
}

func (m *BackendManager) compile(host string, backend *models.Backend) (*localBackend, error) {
	url, err := url.Parse(backend.UpstreamUrl)
	if err != nil {
		return nil, err
	}

	var program *goja.Program = nil
	if backend.ScriptHandler != nil && backend.ScriptHandler.JsScript != "" {
		program, err = goja.Compile("proxy.js", backend.ScriptHandler.JsScript, true)
		if err != nil {
			return nil, err
		}
	}

	policy, err := access.Compile(backend.AccessRules, backend.AccessLevel)
	if err != nil {
		return nil, err
	}

	return &localBackend{m.log, host, backend, url, program, policy}, nil
}

func (m *BackendManager) AddEphemeral(backend Backend) {
	if old, ok := m.ephemeral[backend.Host()]; ok {
		m.log.Infof("Replacing %s backend for %s", old.Type(), old.Host())
	} else {
		m.log.Infof("Adding %s backend for %s", backend.Type(), backend.Host())
	}
	m.ephemeral[backend.Host()] = backend
}

func (m *BackendManager) RemoveEphemeral(b Backend) {
	if old, ok := m.ephemeral[b.Host()]; ok && old == b {
		m.log.Infof("Removing %s backend for %s", b.Type(), b.Host())
		delete(m.ephemeral, b.Host())
	}
}
