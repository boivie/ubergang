package proxy

import (
	"fmt"
	"net/http"
	"net/url"
)

// The query parameter that carries a trampoline token to a backend host.
const trampolineParam = "_ubergang_token"

func (s *Proxy) redirectsigninInvalidSession(w http.ResponseWriter, r *http.Request) {
	redirect := r.URL.Query().Get("rd")
	url := fmt.Sprintf("https://%s/signin?rd=%s", s.config.AdminFqdn, url.QueryEscape(redirect))
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Proxy) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	_, session, err := s.session.Get(r)
	if err != nil {
		s.log.Infof("Failed to find session: %v", err)
		s.redirectsigninInvalidSession(w, r)
		return
	}
	s.updateAccessed <- session

	redirect := r.URL.Query().Get("rd")
	if redirect == "" {
		_, _ = w.Write([]byte("Authorized\n"))
		return
	}

	// Only redirect to backends, as the trampoline token lets whoever receives
	// it sign in as this user.
	u, err := url.Parse(redirect)
	if err != nil || u.Scheme != "https" || u.User != nil {
		s.log.Warnf("Refusing to authorize invalid redirect %q", redirect)
		http.Error(w, "Invalid redirect", http.StatusBadRequest)
		return
	}
	if _, err := s.backends.Lookup(u.Hostname()); err != nil {
		s.log.Warnf("Refusing to authorize redirect to unknown backend %q", u.Hostname())
		http.Error(w, "Invalid redirect", http.StatusBadRequest)
		return
	}

	// Redirect to the trampoline, so that it can set the cookie and bind it to
	// the correct domain.
	token, err := s.session.CreateTrampolineToken(session, u.Hostname())
	if err != nil {
		s.log.Warnf("Failed to create trampoline token: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	q := u.Query()
	q.Set(trampolineParam, token)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}
