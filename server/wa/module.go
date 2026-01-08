package wa

import (
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/models"
	"embed"
	"encoding/json"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type knownAaGuid struct {
	Name      string `json:"name"`
	IconDark  string `json:"icon_dark"`
	IconLight string `json:"icon_light"`
}

type WA struct {
	webAuthn  *webauthn.WebAuthn
	db        *db.DB
	aaguidMap map[string]knownAaGuid
}

var (
	//go:embed "webauthn-data"
	data embed.FS
)

func New(config *models.Configuration, db *db.DB) *WA {
	// Keep the domain, strip the port from `config.AdminFqdn`
	rpId := strings.Split(config.AdminFqdn, ":")[0]

	var aaguidMap map[string]knownAaGuid
	file, err := data.Open("webauthn-data/aaguid.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	json.NewDecoder(file).Decode(&aaguidMap)

	wconfig := &webauthn.Config{
		RPID:                  rpId,
		RPDisplayName:         "ubergang",
		RPOrigins:             []string{"https://" + config.AdminFqdn},
		AttestationPreference: "none",
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: "platform",
			RequireResidentKey:      &[]bool{true}[0],
			ResidentKey:             "required",
			UserVerification:        "required",
		},
	}

	webAuthn, err := webauthn.New(wconfig)
	if err != nil {
		panic(err)
	}
	return &WA{
		webAuthn:  webAuthn,
		db:        db,
		aaguidMap: aaguidMap,
	}
}

func (w *WA) RPID() string {
	return w.webAuthn.Config.RPID
}
