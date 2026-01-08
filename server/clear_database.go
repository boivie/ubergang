package server

import (
	"fmt"
	"log"

	"github.com/charmbracelet/huh"
)

func (s *Server) ClearDatabase() {
	var confirm bool

	err := huh.NewConfirm().
		Title("Clear Database").
		Description("Are you sure you want to clear the database?").
		Value(&confirm).
		Run()

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if confirm {
		s.db.ClearDatabase()
		fmt.Println("Database cleared.")
	} else {
		fmt.Println("Aborted.")
	}
}
