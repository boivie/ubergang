package proxy

import (
	"fmt"
	"net/http"
	"net/url"
)

func (s *Proxy) redirectsigninInvalidSession(w http.ResponseWriter, r *http.Request) {
	redirect := r.URL.Query().Get("rd")
	url := fmt.Sprintf("https://%s/signin?rd=%s", s.config.AdminFqdn, redirect)
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

	// Redirect to the trampoline, so that it can set the cookie and bind it to the correct domain.
	redirect := r.URL.Query().Get("rd")
	if redirect != "" {
		u, err := url.Parse(redirect)
		if err != nil {
			redirect = ""
		} else {
			q := u.Query()
			q.Set("_ubergang_session", s.session.EncodeSessionCookie(session))
			u.RawQuery = q.Encode()
			redirect = u.String()
		}
	}
	if redirect == "" {
		w.Write([]byte("Authorized\n"))
	} else {
		http.Redirect(w, r, redirect, http.StatusFound)
	}
}
