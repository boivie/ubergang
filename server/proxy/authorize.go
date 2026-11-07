package proxy

import (
	"boivie/ubergang/server/session"
	"fmt"
	"net/http"
	"net/url"
)

const trampolineParam = session.TrampolineParam

func (s *Proxy) redirectsigninInvalidSession(w http.ResponseWriter, r *http.Request) {
	redirect := r.URL.Query().Get("rd")
	url := fmt.Sprintf("https://%s/signin?rd=%s", s.config.AdminFqdn, url.QueryEscape(redirect))
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Proxy) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	_, sess, err := s.session.Get(r)
	if err != nil {
		s.log.Infof("Failed to find session: %v", err)
		s.redirectsigninInvalidSession(w, r)
		return
	}
	s.updateAccessed <- sess

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
	target, err := s.session.TrampolineURL(sess, u)
	if err != nil {
		s.log.Warnf("Failed to create trampoline token: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}
