package server

import (
	"boivie/ubergang/server/models"
	"fmt"
	"log"

	"github.com/charmbracelet/huh"
)

func (s *Server) IsConfigured() bool {
	cfg, _ := s.db.GetConfiguration()
	return cfg != nil && cfg.Email != "" && cfg.SiteFqdn != "" && cfg.AdminFqdn != ""
}

func (s *Server) Configure() {
	config, err := s.db.GetConfiguration()
	if err != nil {
		config = &models.Configuration{
			Email:     "",
			SiteFqdn:  "",
			AdminFqdn: "",
		}
	}

	var email = config.Email
	var siteFqdn = config.SiteFqdn

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Configure Server").
				Description("Enter the administrator's email address").
				Placeholder("user@example.com").
				Value(&email).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("email cannot be empty")
					}
					return nil
				}),
			huh.NewInput().
				Description("Site root domain").
				Placeholder("example.com").
				Value(&siteFqdn).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("FQDN cannot be empty")
					}
					return nil
				}),
		),
	)
	err = form.Run()

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	adminFqdn := "account." + siteFqdn

	if err := s.db.UpdateConfiguration(func(old *models.Configuration) (*models.Configuration, error) {
		if old == nil {
			old = &models.Configuration{}
		}
		old.Email = email
		old.SiteFqdn = siteFqdn
		old.AdminFqdn = adminFqdn
		return old, nil
	}); err != nil {
		log.Fatalf("Failed to update configuration: %v", err)
	}
}
