package server

import (
	"boivie/ubergang/server/db"
	"fmt"
	"log"

	"github.com/charmbracelet/huh"
)

func ClearDatabase(db *db.DB) {
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
		if err := db.ClearDatabase(); err != nil {
			log.Fatalf("Failed to clear database: %v", err)
		}
		fmt.Println("Database cleared.")
	} else {
		fmt.Println("Aborted.")
	}
}
