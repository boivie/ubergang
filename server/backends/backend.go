package backends

import (
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"context"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/dop251/goja"
)

type Backend interface {
	Type() string
	Host() string
	Headers() []*models.Header
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	NeedsAuth() bool
	URL() *url.URL
	JsScript() *goja.Program
}

type BackendManager struct {
	db        *db.DB
	log       *log.Log
	ephemeral map[string]Backend
}

func New(db *db.DB, log *log.Log) *BackendManager {
	return &BackendManager{db, log, make(map[string]Backend)}
}

type localBackend struct {
	log     *log.Log
	host    string
	backend *models.Backend
	url     *url.URL
	program *goja.Program
}

func (b *localBackend) Type() string              { return "local" }
func (b *localBackend) Host() string              { return b.host }
func (b *localBackend) Headers() []*models.Header { return b.backend.Headers }
func (b *localBackend) NeedsAuth() bool           { return b.backend.AccessLevel != models.AccessLevel_PUBLIC }
func (b *localBackend) URL() *url.URL             { return b.url }
func (b *localBackend) JsScript() *goja.Program   { return b.program }

func (b *localBackend) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	newAddress := b.url.Host
	if b.url.Port() == "" {
		newAddress = b.url.Host + ":80"
	}

	b.log.Infof("Dialing %s %s / %s", network, address, newAddress)
	return (&net.Dialer{
		Timeout:   2 * time.Second,
		KeepAlive: 5 * time.Second,
		DualStack: true,
	}).DialContext(ctx, network, newAddress)
}

func (m *BackendManager) Lookup(host string) (Backend, error) {
	if strings.Contains(host, ":") {
		host = host[0:strings.Index(host, ":")]
	}

	backend, err := m.db.GetBackend(host)
	if err != nil {
		if b, ok := m.ephemeral[host]; ok {
			return b, nil
		}
		return nil, err
	}

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

	return &localBackend{m.log, host, backend, url, program}, nil
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
