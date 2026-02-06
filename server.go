package main

import (
	"boivie/ubergang/server"
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/models"
	"embed"
	"fmt"

	uglog "boivie/ubergang/server/log"

	flag "github.com/spf13/pflag"

	"os"
)

var (
	//go:embed web/dist
	assets embed.FS
)
var flgDb = flag.String("db", "ubergang.db", "Database file")
var flgConfigure = flag.Bool("configure", false, "Configure server")
var flgAccount = flag.Bool("account", false, "Create account")
var flgTestMode = flag.Bool("test-mode", false, "Test Mode (Only used in integration tests)")

const ADMIN_HOST = "localhost:10443"

func main() {
	flag.Parse()
	fmt.Printf("Starting server, flags: %v, test_mode=%v, db=%s\n", os.Args, *flgTestMode, *flgDb)
	log := uglog.NewLogger(uglog.Fields{})

	log.Infof("Using database at %s", *flgDb)

	db, err := db.New(log, *flgDb)
	if err != nil {
		log.Fatalf("Failed to load database: %v", err)
		os.Exit(1)
	}

	if *flgTestMode {
		err := db.UpdateConfiguration(func(old *models.Configuration) (*models.Configuration, error) {
			if old != nil && old.AdminFqdn != ADMIN_HOST {
				fmt.Println("Test mode is restricted to integration tests. Aborting")
				os.Exit(1)
			}

			config := &models.Configuration{
				Email:        "hello@example.com",
				SiteFqdn:     "example.com",
				AdminFqdn:    ADMIN_HOST,
				IsInTestMode: true,
			}

			return config, nil
		})

		if err != nil {
			fmt.Printf("Failed to update configuration: %v\n", err)
			os.Exit(1)
		}
	}

	if *flgConfigure {
		server.Configure(db)
		return
	}
	if *flgAccount {
		server.CreateAccount(db)
		return
	}

	s := server.NewServer(db, &assets)
	log.Info("Server is starting...")
	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}
