package rest

import (
	"boivie/ubergang/server/auth"
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/mqtt"
	"boivie/ubergang/server/session"
	"boivie/ubergang/server/wa"

	"github.com/gorilla/mux"
)

type ApiModule struct {
	config    *models.Configuration
	log       *log.Log
	db        *db.DB
	session   *session.SessionStore
	auth      *auth.Auth
	webauthn  *wa.WA
	mqttProxy mqtt.ConnectionTracker
}

func New(config *models.Configuration,
	db *db.DB,
	log *log.Log,
	session *session.SessionStore,
	auth *auth.Auth,
	mqttProxy mqtt.ConnectionTracker) *ApiModule {

	return &ApiModule{
		config, log, db, session, auth, wa.New(config, db), mqttProxy,
	}
}

func (a *ApiModule) RegisterEndpoints(r *mux.Router) {
	// Enrolling
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/enroll/start").HandlerFunc(a.handleEnrollStart)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/enroll/finish").HandlerFunc(a.handleEnrollFinish)
	// Signing in
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/signin/start").HandlerFunc(a.handleSigninStart)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/signin/email").HandlerFunc(a.handleSigninEmail)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/signin/webauthn").HandlerFunc(a.handleSigninWebauthn)
	// Signing in, pin flow
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/signin/pin/request").HandlerFunc(a.handleSigninPinRequest)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/signin/pin/poll").HandlerFunc(a.handleSigninPinPoll)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/signin/pin/query").HandlerFunc(a.handleSigninPinQuery)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/signin/pin/confirm").HandlerFunc(a.handleSigninPinConfirm)
	// SSH keys
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/ssh-key/{id}/confirm").HandlerFunc(a.handleGetSshKeyConfirm)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/ssh-key/{id}/confirm").HandlerFunc(a.handlePostSshKeyConfirm)
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/ssh-key/{id}").HandlerFunc(a.handleSshKeyGet)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/ssh-key/{id}").HandlerFunc(a.handleUpdateSshKey)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/ssh-key").HandlerFunc(a.handleSshKeyCreate)
	// Backends
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/backend/{fqdn}").HandlerFunc(a.handleBackendUpdate)
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/backend/{fqdn}").HandlerFunc(a.handleBackendGet)
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/backend").HandlerFunc(a.handleBackendList)
	r.Host(a.config.AdminFqdn).Methods("DELETE").Path("/api/backend/{fqdn}").HandlerFunc(a.handleBackendDelete)
	// MQTT Profiles
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/mqtt-profile/{id}").HandlerFunc(a.handleMqttProfileUpdate)
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/mqtt-profile/{id}").HandlerFunc(a.handleMqttProfileGet)
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/mqtt-profile").HandlerFunc(a.handleMqttProfileList)
	r.Host(a.config.AdminFqdn).Methods("DELETE").Path("/api/mqtt-profile/{id}").HandlerFunc(a.handleMqttProfileDelete)
	// MQTT Clients
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/mqtt-client/{id}").HandlerFunc(a.handleMqttClientUpdate)
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/mqtt-client/{id}").HandlerFunc(a.handleMqttClientGet)
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/mqtt-client").HandlerFunc(a.handleMqttClientList)
	r.Host(a.config.AdminFqdn).Methods("DELETE").Path("/api/mqtt-client/{id}").HandlerFunc(a.handleMqttClientDelete)
	// MQTT Import/Export
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/mqtt/import").HandlerFunc(a.handleMqttImport)
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/mqtt/export").HandlerFunc(a.handleMqttExport)
	// Credentials
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/credential/{id}").HandlerFunc(a.handleCredentialUpdate)
	r.Host(a.config.AdminFqdn).Methods("DELETE").Path("/api/credential/{id}").HandlerFunc(a.handleCredentialDelete)
	// Sessions
	r.Host(a.config.AdminFqdn).Methods("DELETE").Path("/api/session/{id}").HandlerFunc(a.handleSessionDelete)
	// Users
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/user").HandlerFunc(a.handleUserCreate)
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/user").HandlerFunc(a.handleUserList)
	r.Host(a.config.AdminFqdn).Methods("GET").Path("/api/user/{id}").HandlerFunc(a.handleUserGet)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/user/{id}").HandlerFunc(a.handleUserUpdate)
	r.Host(a.config.AdminFqdn).Methods("DELETE").Path("/api/user/{id}").HandlerFunc(a.handleUserDelete)
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/user/{id}/recover").HandlerFunc(a.handleUserRecover)
	// Testing
	r.Host(a.config.AdminFqdn).Methods("POST").Path("/api/testing/setup").HandlerFunc(a.handleTestingSetup)
	// Webauthn Images
	r.Host(a.config.AdminFqdn).Path("/passkey-image/{aaguid}").HandlerFunc(a.webauthn.PasskeyImageHandler)
}

// RegisterBootstrapEndpoints registers API endpoints for bootstrap mode (no Host requirement)
func (a *ApiModule) RegisterBootstrapEndpoints(r *mux.Router) {
	// Bootstrap configuration
	r.Methods("POST").Path("/api/bootstrap/configure").HandlerFunc(a.handleBootstrapConfigure)
	r.Methods("GET").Path("/api/bootstrap/status").HandlerFunc(a.handleBootstrapStatus)
}
