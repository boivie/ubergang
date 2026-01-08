package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"
	"sort"
)

func toMqttProfile(p *models.MqttProfile) api.ApiMqttProfile {
	return api.ApiMqttProfile{
		Id:             p.Id,
		AllowPublish:   p.AllowPublish,
		AllowSubscribe: p.AllowSubscribe,
	}
}

func (s *ApiModule) handleMqttProfileList(w http.ResponseWriter, r *http.Request) {
	_, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	profiles := make([]api.ApiMqttProfile, 0)
	for _, p := range s.db.ListMqttProfiles() {
		profiles = append(profiles, toMqttProfile(p))
	}
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Id < profiles[j].Id
	})

	jsonify(w, api.ApiListMqttProfilesResponse{
		MqttProfiles: profiles,
	})
}
