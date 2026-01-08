package rest

import (
	"boivie/ubergang/server/models"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleMqttClientDelete(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	id := mux.Vars(r)["id"]

	err = s.db.UpdateMqttClient(id, func(old *models.MqttClient) (*models.MqttClient, error) {
		return nil, nil
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
