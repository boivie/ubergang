package main

import (
	"boivie/ubergang/server"
	"embed"

	flag "github.com/spf13/pflag"

	"log"
	"os"
)

var (
	//go:embed web/dist
	assets embed.FS
)
var flgDb = flag.String("db", "ubergang.db", "Database file")
var flgConfigure = flag.Bool("configure", false, "Configure server")
var flgAccount = flag.Bool("account", false, "Create account")
var flgClearDb = flag.Bool("clear-db", false, "Clear database (DANGER!)")
var flgTestMode = flag.Bool("test-mode", false, "Test Mode (Only used in integration tests)")

func main() {
	flag.Parse()
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	s := server.NewServer(*flgDb, &assets)
	if *flgTestMode {
		s.StartTestMode()
		// Fallthrough.
	} else if *flgConfigure {
		s.Configure()
		return
	} else if *flgAccount {
		s.CreateAccount()
		return
	} else if *flgClearDb {
		s.ClearDatabase()
		return
	}

	logger.Println("Server is starting...")
	s.Serve()
}
