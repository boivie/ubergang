package session

import (
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const trampolineTokenExpiry = time.Minute

// The query parameter that carries a trampoline token to a backend host.
const TrampolineParam = "_ubergang_token"

type SessionStore struct {
	log           *log.Log
	db            *db.DB
	sessionCookie string
	sessionExpiry time.Duration
}

func NewSessionStore(log *log.Log, db *db.DB) *SessionStore {
	ss := &SessionStore{
		log:           log,
		db:            db,
		sessionCookie: "__ug_sess",
		sessionExpiry: 10 * 365 * 24 * time.Hour,
	}
	return ss
}

// Get returns the session of the request's cookie, if it's valid and can be
// used on the host that the request was made to.
func (s *SessionStore) Get(r *http.Request) (*models.User, *models.Session, error) {
	c, err := r.Cookie(s.sessionCookie)
	if err != nil {
		return nil, nil, err
	}
	return s.DecodeSessionCookie(c.Value, r.Host, true /*validateSecret*/)
}

func (s *SessionStore) ReuseSession(r *http.Request) (*models.User, *models.Session, error) {
	c, err := r.Cookie(s.sessionCookie)
	if err != nil {
		return nil, nil, err
	}
	return s.DecodeSessionCookie(c.Value, r.Host, false /*validateSecret*/)
}

func (s *SessionStore) GetWithUser(r *http.Request) (*models.User, *models.Session, error) {
	c, err := r.Cookie(s.sessionCookie)
	if err != nil {
		return nil, nil, err
	}
	return s.DecodeSessionCookie(c.Value, r.Host, true /*validateSecret*/)
}

func (s *SessionStore) GetAndValidate(w http.ResponseWriter, r *http.Request) (*models.User, *models.Session, error) {
	user, session, err := s.Get(r)
	if err != nil {
		s.log.Warnf("Failed to authenticate: %v", err)
		http.Error(w, "Not authorized", http.StatusForbidden)
		return nil, nil, err
	}
	return user, session, err
}

func (s *SessionStore) CreateSessionCookie(session *models.Session) *http.Cookie {
	expires := time.Now().Add(s.sessionExpiry)
	if session.ExpiresAt != nil {
		expires = session.ExpiresAt.AsTime()
	}
	return &http.Cookie{
		Name:    s.sessionCookie,
		Path:    "/",
		Value:   s.EncodeSessionCookie(session),
		Expires: expires,
		Secure:  true,
	}
}

func (s *SessionStore) EncodeSessionCookie(session *models.Session) string {
	return fmt.Sprintf("%s:%s", session.Id, session.Secret)
}

// StripSessionCookie removes the session cookie from the Cookie headers in
// `h`, keeping all other cookies as sent. Backends must never see it, as it's
// valid on the admin host as well.
func (s *SessionStore) StripSessionCookie(h http.Header) {
	var kept []string
	for _, line := range h.Values("Cookie") {
		var parts []string
		for _, part := range strings.Split(line, ";") {
			part = strings.TrimSpace(part)
			name, _, _ := strings.Cut(part, "=")
			if part != "" && name != s.sessionCookie {
				parts = append(parts, part)
			}
		}
		if len(parts) > 0 {
			kept = append(kept, strings.Join(parts, "; "))
		}
	}
	h.Del("Cookie")
	for _, line := range kept {
		h.Add("Cookie", line)
	}
}

// normalizeHost lowercases `host` and strips any port from it.
func normalizeHost(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.ToLower(host)
}

// CreateTrampolineToken returns a single-use token that can be exchanged for
// the session's cookie on `host` (only), within a short time.
func (s *SessionStore) CreateTrampolineToken(session *models.Session, host string) (string, error) {
	token := rand.Text()
	err := s.db.StoreTrampolineToken(token, &models.TrampolineToken{
		SessionId: session.Id,
		Host:      normalizeHost(host),
		ExpiresAt: timestamppb.New(time.Now().Add(trampolineTokenExpiry)),
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

// TrampolineURL returns `redirect` with a trampoline token added, which sets
// the session's cookie on the redirect's host when followed.
func (s *SessionStore) TrampolineURL(session *models.Session, redirect *url.URL) (string, error) {
	token, err := s.CreateTrampolineToken(session, redirect.Hostname())
	if err != nil {
		return "", err
	}
	u := *redirect
	q := u.Query()
	q.Set(TrampolineParam, token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ConsumeTrampolineToken exchanges a token created by CreateTrampolineToken
// for its session, if it was issued for `host` and hasn't expired. The token
// can't be used again, whether this succeeds or not.
func (s *SessionStore) ConsumeTrampolineToken(token, host string) (*models.User, *models.Session, error) {
	t, err := s.db.ConsumeTrampolineToken(token)
	if err != nil {
		return nil, nil, err
	}
	if t.Host != normalizeHost(host) {
		return nil, nil, fmt.Errorf("trampoline token issued for %s, used on %s", t.Host, host)
	}
	if t.ExpiresAt.AsTime().Before(time.Now()) {
		return nil, nil, errors.New("trampoline token has expired")
	}
	user, session, err := s.db.GetSession(t.SessionId)
	if err != nil {
		return nil, nil, err
	}
	if err := usableOn(session, host); err != nil {
		return nil, nil, err
	}
	return user, session, nil
}

// usableOn returns an error if `session` can't be used on `host`, because it
// has expired or because it's restricted to another host.
func usableOn(session *models.Session, host string) error {
	if session.ExpiresAt != nil && !time.Now().Before(session.ExpiresAt.AsTime()) {
		return fmt.Errorf("session %s has expired", session.Id)
	}
	if session.RestrictedHost != "" && session.RestrictedHost != normalizeHost(host) {
		return fmt.Errorf("session %s is restricted to %s, used on %s", session.Id, session.RestrictedHost, host)
	}
	return nil
}

// DecodeSessionCookie returns the session of a cookie's value, if it can be
// used on `host`.
func (s *SessionStore) DecodeSessionCookie(value, host string, validateSecret bool) (*models.User, *models.Session, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return nil, nil, errors.New("invalid cookie structure")
	}
	user, session, err := s.db.GetSession(parts[0])
	if err != nil {
		return nil, nil, err
	}

	if validateSecret && session.Secret != parts[1] {
		return nil, nil, errors.New("invalid session secret")
	}
	if err := usableOn(session, host); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}
