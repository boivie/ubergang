package cert

import (
	"boivie/ubergang/server/db"
	"context"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type CertMagicStorage struct {
	db *db.DB
	// For single-instance locking
	locksMu sync.Mutex
	locks   map[string]*sync.Mutex
}

var (
	certUpdatesTotalMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ubergang_cert_updates_total",
		Help: "The total number of certification updates",
	}, []string{"cert"})
	certLastUpdatedMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ubergang_cert_last_updated",
		Help: "The timestamp when the certificate was last updated",
	}, []string{"cert"})
)

func NewCertMagicStorage(db *db.DB) certmagic.Storage {
	return &CertMagicStorage{
		db:    db,
		locks: make(map[string]*sync.Mutex),
	}
}

// Store puts value at key
func (s *CertMagicStorage) Store(ctx context.Context, key string, value []byte) error {
	// Track certificate updates in metrics (skip ACME account keys with "+")
	if !strings.Contains(key, "+") {
		certUpdatesTotalMetric.WithLabelValues(key).Inc()
		certLastUpdatedMetric.WithLabelValues(key).SetToCurrentTime()
	}
	return s.db.UpdateCert(key, value)
}

// Load retrieves the value at key
func (s *CertMagicStorage) Load(ctx context.Context, key string) ([]byte, error) {
	data, err := s.db.GetCert(key)
	if err != nil {
		return nil, fs.ErrNotExist
	}
	return data, nil
}

// Delete deletes the named key
func (s *CertMagicStorage) Delete(ctx context.Context, key string) error {
	// For prefix deletion, we need to delete all keys with this prefix
	if strings.HasSuffix(key, "/") || s.isPrefix(key) {
		return s.deletePrefix(key)
	}
	return s.db.DeleteCert(key)
}

// deletePrefix removes all keys starting with the given prefix
func (s *CertMagicStorage) deletePrefix(prefix string) error {
	return s.db.DeleteCertsByPrefix(prefix)
}

// isPrefix checks if the key is a prefix of other keys
func (s *CertMagicStorage) isPrefix(key string) bool {
	keys, err := s.db.ListCertKeys()
	if err != nil {
		return false
	}

	for _, k := range keys {
		if strings.HasPrefix(k, key+"/") {
			return true
		}
	}
	return false
}

// Exists returns true if the key exists
func (s *CertMagicStorage) Exists(ctx context.Context, key string) bool {
	// Check if it's a file (exact key match)
	_, err := s.db.GetCert(key)
	if err == nil {
		return true
	}

	// Check if it's a directory (prefix of other keys)
	return s.isPrefix(key)
}

// List returns all keys in the given path
func (s *CertMagicStorage) List(ctx context.Context, path string, recursive bool) ([]string, error) {
	allKeys, err := s.db.ListCertKeys()
	if err != nil {
		return nil, err
	}

	var keys []string
	seenDirs := make(map[string]bool)

	for _, keyName := range allKeys {
		// Filter by path prefix
		if path != "" && !strings.HasPrefix(keyName, path+"/") && keyName != path {
			continue
		}

		// Remove the path prefix for relative keys
		relKey := keyName
		if path != "" {
			relKey = strings.TrimPrefix(keyName, path+"/")
		}

		if recursive {
			// In recursive mode, include all keys under the path
			keys = append(keys, keyName)
		} else {
			// In non-recursive mode, only include direct children
			parts := strings.Split(relKey, "/")
			if len(parts) == 1 {
				// Direct file
				keys = append(keys, keyName)
			} else if len(parts) > 1 {
				// Directory - add only the first component
				dirName := path
				if path != "" {
					dirName = path + "/" + parts[0]
				} else {
					dirName = parts[0]
				}
				if !seenDirs[dirName] {
					keys = append(keys, dirName)
					seenDirs[dirName] = true
				}
			}
		}
	}

	if len(keys) == 0 {
		return nil, fs.ErrNotExist
	}

	return keys, nil
}

// Stat returns information about key
func (s *CertMagicStorage) Stat(ctx context.Context, key string) (certmagic.KeyInfo, error) {
	data, err := s.db.GetCert(key)
	if err == nil {
		// It's a file
		return certmagic.KeyInfo{
			Key:        key,
			Modified:   time.Now(), // BoltDB doesn't track modification time
			Size:       int64(len(data)),
			IsTerminal: true,
		}, nil
	}

	// Check if it's a directory
	if s.isPrefix(key) {
		return certmagic.KeyInfo{
			Key:        key,
			Modified:   time.Time{},
			Size:       0,
			IsTerminal: false,
		}, nil
	}

	return certmagic.KeyInfo{}, fs.ErrNotExist
}

// Lock acquires a lock for the given name
func (s *CertMagicStorage) Lock(ctx context.Context, name string) error {
	s.locksMu.Lock()
	mu, exists := s.locks[name]
	if !exists {
		mu = &sync.Mutex{}
		s.locks[name] = mu
	}
	s.locksMu.Unlock()

	// Try to acquire the lock with context support
	acquired := make(chan struct{})
	go func() {
		mu.Lock()
		close(acquired)
	}()

	select {
	case <-acquired:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("lock acquisition cancelled: %w", ctx.Err())
	}
}

// Unlock releases the lock for the given name
func (s *CertMagicStorage) Unlock(ctx context.Context, name string) error {
	s.locksMu.Lock()
	mu, exists := s.locks[name]
	s.locksMu.Unlock()

	if !exists {
		return fmt.Errorf("lock not found: %s", name)
	}

	mu.Unlock()
	return nil
}
