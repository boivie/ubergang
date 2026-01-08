package rest

import (
	"boivie/ubergang/server/models"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleMqttProfileDelete(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	id := mux.Vars(r)["id"]

	err = s.db.UpdateMqttProfile(id, func(old *models.MqttProfile) (*models.MqttProfile, error) {
		return nil, nil
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
