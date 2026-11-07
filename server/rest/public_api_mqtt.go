package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/auth"
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/models"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/gorilla/mux"
)

var (
	errPublicNotFound      = errors.New("not found")
	errPublicAlreadyExists = errors.New("already exists")
)

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func toPublicMqttProfile(p *models.MqttProfile) api.PublicMqttProfile {
	return api.PublicMqttProfile{
		Id:             p.Id,
		AllowPublish:   orEmpty(p.AllowPublish),
		AllowSubscribe: orEmpty(p.AllowSubscribe),
	}
}

func toPublicMqttClient(c *models.MqttClient, key *models.ApiKey) api.PublicMqttClient {
	ret := api.PublicMqttClient{
		Id:        c.Id,
		ProfileId: c.ProfileId,
		Values:    c.Values,
	}
	if ret.Values == nil {
		ret.Values = map[string]string{}
	}
	if auth.ApiKeyHasScope(key, models.ApiKeyScope_API_KEY_SCOPE_WRITE) {
		ret.Password = c.Password
	}
	return ret
}

// disconnectMqttClientsWithProfile makes clients using `profileId` reconnect,
// so that a changed profile takes effect.
func (s *ApiModule) disconnectMqttClientsWithProfile(profileId string) {
	for _, c := range s.db.ListMqttClients() {
		if c.ProfileId == profileId {
			s.mqttProxy.DisconnectClient(c.Id)
		}
	}
}

// MQTT profiles

func (s *ApiModule) handlePublicMqttProfileList(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	profiles := make([]api.PublicMqttProfile, 0)
	for _, p := range s.db.ListMqttProfiles() {
		profiles = append(profiles, toPublicMqttProfile(p))
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Id < profiles[j].Id })
	writePublicJson(w, http.StatusOK, api.PublicListMqttProfilesResponse{Profiles: profiles})
}

func (s *ApiModule) handlePublicMqttProfileGet(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	id := mux.Vars(r)["id"]
	profile, err := s.db.GetMqttProfile(id)
	if err != nil {
		writePublicError(w, http.StatusNotFound, publicErrNotFound, "MQTT profile not found")
		return
	}
	writePublicJson(w, http.StatusOK, toPublicMqttProfile(profile))
}

func (s *ApiModule) handlePublicMqttProfileCreate(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	var req api.PublicCreateMqttProfileRequest
	if !parsePublicJson(w, r, &req) {
		return
	}
	if err := validatePublicId("id", req.Id); err != nil {
		writePublicError(w, http.StatusBadRequest, publicErrBadRequest, err.Error())
		return
	}

	profile := &models.MqttProfile{
		Id:             req.Id,
		AllowPublish:   orEmpty(req.AllowPublish),
		AllowSubscribe: orEmpty(req.AllowSubscribe),
	}
	err := s.db.UpdateMqttProfile(req.Id, func(old *models.MqttProfile) (*models.MqttProfile, error) {
		if old != nil {
			return nil, errPublicAlreadyExists
		}
		return profile, nil
	})
	if errors.Is(err, errPublicAlreadyExists) {
		writePublicError(w, http.StatusConflict, publicErrConflict,
			fmt.Sprintf("MQTT profile %q already exists", req.Id))
		return
	} else if err != nil {
		s.log.Warnf("Failed to create MQTT profile %s: %v", req.Id, err)
		writePublicError(w, http.StatusInternalServerError, publicErrInternal, "Failed to create MQTT profile")
		return
	}
	s.log.Infof("API key %s created MQTT profile %s", key.Id, req.Id)
	w.Header().Set("Location", publicApiPrefix+"/mqtt/profiles/"+url.PathEscape(req.Id))
	writePublicJson(w, http.StatusCreated, toPublicMqttProfile(profile))
}

func (s *ApiModule) handlePublicMqttProfileUpdate(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	id := mux.Vars(r)["id"]
	var req api.PublicUpdateMqttProfileRequest
	if !parsePublicJson(w, r, &req) {
		return
	}

	var updated *models.MqttProfile
	err := s.db.UpdateMqttProfile(id, func(old *models.MqttProfile) (*models.MqttProfile, error) {
		if old == nil {
			return nil, errPublicNotFound
		}
		if req.AllowPublish != nil {
			old.AllowPublish = orEmpty(*req.AllowPublish)
		}
		if req.AllowSubscribe != nil {
			old.AllowSubscribe = orEmpty(*req.AllowSubscribe)
		}
		updated = old
		return old, nil
	})
	if errors.Is(err, errPublicNotFound) {
		writePublicError(w, http.StatusNotFound, publicErrNotFound, "MQTT profile not found")
		return
	} else if err != nil {
		s.log.Warnf("Failed to update MQTT profile %s: %v", id, err)
		writePublicError(w, http.StatusInternalServerError, publicErrInternal, "Failed to update MQTT profile")
		return
	}
	s.log.Infof("API key %s updated MQTT profile %s", key.Id, id)
	if req.AllowPublish != nil || req.AllowSubscribe != nil {
		s.disconnectMqttClientsWithProfile(id)
	}
	writePublicJson(w, http.StatusOK, toPublicMqttProfile(updated))
}

func (s *ApiModule) handlePublicMqttProfileDelete(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	id := mux.Vars(r)["id"]
	err := s.db.UpdateMqttProfile(id, func(old *models.MqttProfile) (*models.MqttProfile, error) {
		if old == nil {
			return nil, errPublicNotFound
		}
		return nil, nil
	})
	if errors.Is(err, errPublicNotFound) {
		writePublicError(w, http.StatusNotFound, publicErrNotFound, "MQTT profile not found")
		return
	} else if errors.Is(err, db.ErrMqttProfileInUse) {
		writePublicError(w, http.StatusConflict, publicErrConflict, err.Error())
		return
	} else if err != nil {
		s.log.Warnf("Failed to delete MQTT profile %s: %v", id, err)
		writePublicError(w, http.StatusInternalServerError, publicErrInternal, "Failed to delete MQTT profile")
		return
	}
	s.log.Infof("API key %s deleted MQTT profile %s", key.Id, id)
	w.WriteHeader(http.StatusNoContent)
}

// MQTT clients

func (s *ApiModule) handlePublicMqttClientList(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	clients := make([]api.PublicMqttClient, 0)
	for _, c := range s.db.ListMqttClients() {
		clients = append(clients, toPublicMqttClient(c, key))
	}
	sort.Slice(clients, func(i, j int) bool { return clients[i].Id < clients[j].Id })
	writePublicJson(w, http.StatusOK, api.PublicListMqttClientsResponse{Clients: clients})
}

func (s *ApiModule) handlePublicMqttClientGet(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	id := mux.Vars(r)["id"]
	client, err := s.db.GetMqttClient(id)
	if err != nil {
		writePublicError(w, http.StatusNotFound, publicErrNotFound, "MQTT client not found")
		return
	}
	writePublicJson(w, http.StatusOK, toPublicMqttClient(client, key))
}

func writeMqttProfileNotFound(w http.ResponseWriter, profileId string) {
	writePublicError(w, http.StatusUnprocessableEntity, "profile_not_found",
		fmt.Sprintf("MQTT profile %q doesn't exist", profileId))
}

func (s *ApiModule) handlePublicMqttClientCreate(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	var req api.PublicCreateMqttClientRequest
	if !parsePublicJson(w, r, &req) {
		return
	}
	if err := validatePublicId("id", req.Id); err != nil {
		writePublicError(w, http.StatusBadRequest, publicErrBadRequest, err.Error())
		return
	}
	if req.ProfileId == "" {
		writePublicError(w, http.StatusBadRequest, publicErrBadRequest, "profile_id is required")
		return
	}
	if req.Password == "" {
		writePublicError(w, http.StatusBadRequest, publicErrBadRequest, "password is required")
		return
	}

	client := &models.MqttClient{
		Id:        req.Id,
		ProfileId: req.ProfileId,
		Password:  req.Password,
		Values:    req.Values,
	}
	err := s.db.UpdateMqttClient(req.Id, func(old *models.MqttClient) (*models.MqttClient, error) {
		if old != nil {
			return nil, errPublicAlreadyExists
		}
		return client, nil
	})
	if errors.Is(err, errPublicAlreadyExists) {
		writePublicError(w, http.StatusConflict, publicErrConflict,
			fmt.Sprintf("MQTT client %q already exists", req.Id))
		return
	} else if errors.Is(err, db.ErrMqttProfileNotFound) {
		writeMqttProfileNotFound(w, req.ProfileId)
		return
	} else if err != nil {
		s.log.Warnf("Failed to create MQTT client %s: %v", req.Id, err)
		writePublicError(w, http.StatusInternalServerError, publicErrInternal, "Failed to create MQTT client")
		return
	}
	s.log.Infof("API key %s created MQTT client %s", key.Id, req.Id)
	w.Header().Set("Location", publicApiPrefix+"/mqtt/clients/"+url.PathEscape(req.Id))
	writePublicJson(w, http.StatusCreated, toPublicMqttClient(client, key))
}

func (s *ApiModule) handlePublicMqttClientUpdate(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	id := mux.Vars(r)["id"]
	var req api.PublicUpdateMqttClientRequest
	if !parsePublicJson(w, r, &req) {
		return
	}
	if req.ProfileId != nil && *req.ProfileId == "" {
		writePublicError(w, http.StatusBadRequest, publicErrBadRequest, "profile_id must not be empty")
		return
	}
	if req.Password != nil && *req.Password == "" {
		writePublicError(w, http.StatusBadRequest, publicErrBadRequest, "password must not be empty")
		return
	}

	var updated *models.MqttClient
	err := s.db.UpdateMqttClient(id, func(old *models.MqttClient) (*models.MqttClient, error) {
		if old == nil {
			return nil, errPublicNotFound
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
		updated = old
		return old, nil
	})
	if errors.Is(err, errPublicNotFound) {
		writePublicError(w, http.StatusNotFound, publicErrNotFound, "MQTT client not found")
		return
	} else if errors.Is(err, db.ErrMqttProfileNotFound) {
		writeMqttProfileNotFound(w, *req.ProfileId)
		return
	} else if err != nil {
		s.log.Warnf("Failed to update MQTT client %s: %v", id, err)
		writePublicError(w, http.StatusInternalServerError, publicErrInternal, "Failed to update MQTT client")
		return
	}
	s.log.Infof("API key %s updated MQTT client %s", key.Id, id)
	if req.ProfileId != nil || req.Password != nil {
		// Make the client reconnect with its new credentials and permissions.
		s.mqttProxy.DisconnectClient(id)
	}
	writePublicJson(w, http.StatusOK, toPublicMqttClient(updated, key))
}

func (s *ApiModule) handlePublicMqttClientDelete(w http.ResponseWriter, r *http.Request, key *models.ApiKey) {
	id := mux.Vars(r)["id"]
	err := s.db.UpdateMqttClient(id, func(old *models.MqttClient) (*models.MqttClient, error) {
		if old == nil {
			return nil, errPublicNotFound
		}
		return nil, nil
	})
	if errors.Is(err, errPublicNotFound) {
		writePublicError(w, http.StatusNotFound, publicErrNotFound, "MQTT client not found")
		return
	} else if err != nil {
		s.log.Warnf("Failed to delete MQTT client %s: %v", id, err)
		writePublicError(w, http.StatusInternalServerError, publicErrInternal, "Failed to delete MQTT client")
		return
	}
	s.log.Infof("API key %s deleted MQTT client %s", key.Id, id)
	s.mqttProxy.DisconnectClient(id)
	w.WriteHeader(http.StatusNoContent)
}
