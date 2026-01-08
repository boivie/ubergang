package rest

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleBackendGet(w http.ResponseWriter, r *http.Request) {
	_, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	fqdn := strings.ToLower(mux.Vars(r)["fqdn"])

	be, err := s.db.GetBackend(fqdn)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	jsonify(w, ToBackend(be))
}
