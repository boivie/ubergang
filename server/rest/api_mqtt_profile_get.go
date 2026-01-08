package rest

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleMqttProfileGet(w http.ResponseWriter, r *http.Request) {
	_, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	id := mux.Vars(r)["id"]

	profile, err := s.db.GetMqttProfile(id)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	jsonify(w, toMqttProfile(profile))
}
