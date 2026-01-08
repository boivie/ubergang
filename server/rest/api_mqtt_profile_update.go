package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *ApiModule) handleMqttProfileUpdate(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	var req api.ApiUpdateMqttProfileRequest
	err = parseJsonRequest(w, r, &req)
	if err != nil {
		return
	}

	id := mux.Vars(r)["id"]

	err = s.db.UpdateMqttProfile(id, func(old *models.MqttProfile) (*models.MqttProfile, error) {
		if old == nil {
			old = &models.MqttProfile{
				Id: id,
			}
		}
		if req.AllowPublish != nil {
			old.AllowPublish = *req.AllowPublish
		}
		if req.AllowSubscribe != nil {
			old.AllowSubscribe = *req.AllowSubscribe
		}
		return old, nil
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	jsonify(w, api.ApiUpdateMqttProfileResponse{})
}
