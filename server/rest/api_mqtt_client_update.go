package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleMqttClientUpdate(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	var req api.ApiUpdateMqttClientRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	id := mux.Vars(r)["id"]

	err = s.db.UpdateMqttClient(id, func(old *models.MqttClient) (*models.MqttClient, error) {
		if old == nil {
			old = &models.MqttClient{
				Id: id,
			}
		}
		if req.Id != nil {
			old.Id = *req.Id
		}
		if req.ProfileId != nil {
			old.ProfileId = *req.ProfileId
		}
		if req.Password != nil {
			old.Password = *req.Password
		}
		if req.Values != nil {
			old.Values = *req.Values
		}
		return old, nil
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	jsonify(w, api.ApiUpdateMqttClientResponse{})
}
