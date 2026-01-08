package rest

import (
	"boivie/ubergang/server/models"
	"fmt"
	"net/http"

	"gopkg.in/yaml.v3"
)

// YAML structures for import/export
type YamlMqttClient struct {
	Name     string            `yaml:"name"`
	Password string            `yaml:"password"`
	Profile  string            `yaml:"profile"`
	Values   map[string]string `yaml:"values,omitempty"`
}

type YamlMqttProfile struct {
	Name           string   `yaml:"name"`
	AllowPublish   []string `yaml:"allow_publish"`
	AllowSubscribe []string `yaml:"allow_subscribe"`
}

type YamlMqttConfig struct {
	Clients  []YamlMqttClient  `yaml:"clients"`
	Profiles []YamlMqttProfile `yaml:"profiles"`
}

func (s *ApiModule) handleMqttExport(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	// Fetch all profiles
	profiles := s.db.ListMqttProfiles()
	yamlProfiles := make([]YamlMqttProfile, 0, len(profiles))
	for _, p := range profiles {
		yamlProfiles = append(yamlProfiles, YamlMqttProfile{
			Name:           p.Id,
			AllowPublish:   p.AllowPublish,
			AllowSubscribe: p.AllowSubscribe,
		})
	}

	// Fetch all clients
	clients := s.db.ListMqttClients()
	yamlClients := make([]YamlMqttClient, 0, len(clients))
	for _, c := range clients {
		yamlClients = append(yamlClients, YamlMqttClient{
			Name:     c.Id,
			Password: c.Password,
			Profile:  c.ProfileId,
			Values:   c.Values,
		})
	}

	config := YamlMqttConfig{
		Clients:  yamlClients,
		Profiles: yamlProfiles,
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		s.log.Error("Failed to marshal MQTT config to YAML", "error", err)
		http.Error(w, "Failed to export configuration", http.StatusInternalServerError)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=mqtt-config.yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(yamlData)
}

func (s *ApiModule) handleMqttImport(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	if !user.IsAdmin {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	// Parse YAML from request body
	var config YamlMqttConfig
	decoder := yaml.NewDecoder(r.Body)
	if err := decoder.Decode(&config); err != nil {
		s.log.Error("Failed to parse YAML", "error", err)
		http.Error(w, fmt.Sprintf("Failed to parse YAML: %v", err), http.StatusBadRequest)
		return
	}

	// Validate that all referenced profiles exist or will be created
	profilesMap := make(map[string]bool)
	for _, p := range config.Profiles {
		if p.Name == "" {
			http.Error(w, "Profile name cannot be empty", http.StatusBadRequest)
			return
		}
		profilesMap[p.Name] = true
	}

	// Validate clients reference valid profiles
	for _, c := range config.Clients {
		if c.Name == "" {
			http.Error(w, "Client name cannot be empty", http.StatusBadRequest)
			return
		}
		if c.Profile == "" {
			http.Error(w, fmt.Sprintf("Client '%s' must have a profile", c.Name), http.StatusBadRequest)
			return
		}
		if !profilesMap[c.Profile] {
			// Check if profile exists in database
			if _, err := s.db.GetMqttProfile(c.Profile); err != nil {
				http.Error(w, fmt.Sprintf("Client '%s' references unknown profile '%s'", c.Name, c.Profile), http.StatusBadRequest)
				return
			}
		}
	}

	// Import profiles first
	for _, p := range config.Profiles {
		err := s.db.UpdateMqttProfile(p.Name, func(old *models.MqttProfile) (*models.MqttProfile, error) {
			return &models.MqttProfile{
				Id:             p.Name,
				AllowPublish:   p.AllowPublish,
				AllowSubscribe: p.AllowSubscribe,
			}, nil
		})
		if err != nil {
			s.log.Error("Failed to import profile", "profile", p.Name, "error", err)
			http.Error(w, fmt.Sprintf("Failed to import profile '%s': %v", p.Name, err), http.StatusInternalServerError)
			return
		}
	}

	// Import clients
	for _, c := range config.Clients {
		err := s.db.UpdateMqttClient(c.Name, func(old *models.MqttClient) (*models.MqttClient, error) {
			return &models.MqttClient{
				Id:        c.Name,
				ProfileId: c.Profile,
				Password:  c.Password,
				Values:    c.Values,
			}, nil
		})
		if err != nil {
			s.log.Error("Failed to import client", "client", c.Name, "error", err)
			http.Error(w, fmt.Sprintf("Failed to import client '%s': %v", c.Name, err), http.StatusInternalServerError)
			return
		}
	}

	s.log.Info("Successfully imported MQTT configuration",
		"profiles", len(config.Profiles),
		"clients", len(config.Clients))

	w.WriteHeader(http.StatusOK)
	jsonify(w, map[string]interface{}{
		"success":        true,
		"profiles_count": len(config.Profiles),
		"clients_count":  len(config.Clients),
	})
}
