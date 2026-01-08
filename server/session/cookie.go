package session

import (
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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

func (s *SessionStore) Get(r *http.Request) (*models.User, *models.Session, error) {
	c, err := r.Cookie(s.sessionCookie)
	if err != nil {
		return nil, nil, err
	}
	return s.DecodeSessionCookie(c.Value, true /*validateSecret*/)
}

func (s *SessionStore) ReuseSession(r *http.Request) (*models.User, *models.Session, error) {
	c, err := r.Cookie(s.sessionCookie)
	if err != nil {
		return nil, nil, err
	}
	return s.DecodeSessionCookie(c.Value, false /*validateSecret*/)
}

func (s *SessionStore) GetWithUser(r *http.Request) (*models.User, *models.Session, error) {
	c, err := r.Cookie(s.sessionCookie)
	if err != nil {
		return nil, nil, err
	}
	return s.DecodeSessionCookie(c.Value, true /*validateSecret*/)
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
	return &http.Cookie{
		Name:    s.sessionCookie,
		Path:    "/",
		Value:   s.EncodeSessionCookie(session),
		Expires: time.Now().Add(s.sessionExpiry),
		Secure:  true,
	}
}

func (s *SessionStore) EncodeSessionCookie(session *models.Session) string {
	return fmt.Sprintf("%s:%s", session.Id, session.Secret)
}

func (s *SessionStore) DecodeSessionCookie(value string, validateSecret bool) (*models.User, *models.Session, error) {
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

	return user, session, nil
}
