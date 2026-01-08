package rest

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleBackendDelete(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	fqdn := strings.ToLower(mux.Vars(r)["fqdn"])

	err = s.db.DeleteBackend(fqdn)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
