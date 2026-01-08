package rest

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleMqttClientGet(w http.ResponseWriter, r *http.Request) {
	_, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	id := mux.Vars(r)["id"]

	client, err := s.db.GetMqttClient(id)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	jsonify(w, toMqttClient(client,
		s.mqttProxy.GetActiveConnections()[client.Id]))
}
