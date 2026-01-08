package server

import (
	"boivie/ubergang/server/models"
	"fmt"
	"os"
)

const ADMIN_HOST = "localhost:10443"

func (s *Server) StartTestMode() {
	s.db.UpdateConfiguration(func(old *models.Configuration) (*models.Configuration, error) {
		if old != nil && old.AdminFqdn != ADMIN_HOST {
			fmt.Println("Test mode is restricted to integration tests. Aborting")
			os.Exit(1)
		}

		s.config = &models.Configuration{
			Email:        "hello@example.com",
			SiteFqdn:     "example.com",
			AdminFqdn:    ADMIN_HOST,
			IsInTestMode: true,
		}

		return s.config, nil
	})
}
