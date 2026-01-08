package server

import (
	"fmt"
	"log"

	"github.com/charmbracelet/huh"
)

func (s *Server) CreateAccount() {
	var email string
	var admin bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Create User Account").
				Description("Enter the user's email address").
				Placeholder("user@example.com").
				Value(&email).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("email cannot be empty")
					}
					return nil
				}),
			huh.NewConfirm().
				Title("Admin privileges").
				Description("Should this user have admin privileges?").
				Value(&admin),
		),
	)
	err := form.Run()

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if email == "" {
		fmt.Println("Aborted.")
		return
	}

	user, token, err := s.auth.CreateUser(email, email, admin, nil)
	if err != nil {
		log.Fatalf("Error creating user: %v", err)
	}

	fmt.Printf("Success! %s has been created: https://%s/signin/%s\n", user.Email, s.config.AdminFqdn, token)
}
