package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/auth"
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/mqtt"
	"boivie/ubergang/server/session"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/descope/virtualwebauthn"
	"github.com/gorilla/mux"
	"github.com/samber/lo"
)

type Fixture struct {
	Session *session.SessionStore
	Auth    *auth.Auth
	router  *mux.Router
	Db      *db.DB
}

type FakeMqttConnectionTracker struct{}

func (f *FakeMqttConnectionTracker) GetActiveConnections() map[string]*mqtt.ClientConnectionState {
	return make(map[string]*mqtt.ClientConnectionState)
}

func (f *FakeMqttConnectionTracker) DisconnectClient(clientId string) {}

func CreateFixture(t *testing.T) *Fixture {
	config := &models.Configuration{
		AdminFqdn: "test.example.com",
	}

	log := log.NewLogger(log.Fields{})
	db, err := db.New(log, path.Join(t.TempDir(), "test.db"))
	if err != nil {
		panic(err)
	}
	session := session.NewSessionStore(log, db)
	auth := auth.New(log, db)
	if err != nil {
		panic(err)
	}
	api := New(config, db, log, session, auth, &FakeMqttConnectionTracker{})
	router := mux.NewRouter()
	api.RegisterEndpoints(router)
	return &Fixture{
		session,
		auth,
		router,
		db}
}

func (f *Fixture) getUser(cookie *http.Cookie, id string) *api.ApiUser {
	resp := &api.ApiUser{}
	f.request("GET", "/api/user/"+id, nil, cookie, resp)
	return resp
}

func (f *Fixture) CreateBackend(cookie *http.Cookie, backend *api.ApiBackend) *httptest.ResponseRecorder {
	req := &api.ApiUpdateBackendRequest{
		UpstreamUrl: &backend.UpstreamUrl,
		Headers:     &backend.Headers,
		JsScript:    backend.JsScript,
	}
	if backend.AccessLevel != "" {
		req.AccessLevel = &backend.AccessLevel
	}
	resp := &api.ApiUpdateBackendResponse{}
	return f.request("POST", "/api/backend/"+backend.Fqdn, req, cookie, resp)
}

func (f *Fixture) ListBackends(cookie *http.Cookie) []api.ApiBackend {
	resp := &api.ApiListBackendsResponse{}
	f.request("GET", "/api/backend", nil, cookie, resp)
	return resp.Backends
}

func (f *Fixture) CreateMqttProfile(cookie *http.Cookie, profile *api.ApiMqttProfile) *httptest.ResponseRecorder {
	req := &api.ApiUpdateMqttProfileRequest{
		AllowPublish:   &profile.AllowPublish,
		AllowSubscribe: &profile.AllowSubscribe,
	}
	resp := &api.ApiUpdateMqttProfileResponse{}
	return f.request("POST", "/api/mqtt-profile/"+profile.Id, req, cookie, resp)
}

func (f *Fixture) ListMqttProfiles(cookie *http.Cookie) []api.ApiMqttProfile {
	resp := &api.ApiListMqttProfilesResponse{}
	f.request("GET", "/api/mqtt-profile", nil, cookie, resp)
	return resp.MqttProfiles
}

func (f *Fixture) GetMqttProfile(cookie *http.Cookie, id string) *api.ApiMqttProfile {
	resp := &api.ApiMqttProfile{}
	f.request("GET", "/api/mqtt-profile/"+id, nil, cookie, resp)
	return resp
}

func (f *Fixture) DeleteMqttProfile(cookie *http.Cookie, id string) *httptest.ResponseRecorder {
	return f.request("DELETE", "/api/mqtt-profile/"+id, nil, cookie, nil)
}

func (f *Fixture) CreateMqttClient(cookie *http.Cookie, client *api.ApiMqttClient) *httptest.ResponseRecorder {
	req := &api.ApiUpdateMqttClientRequest{
		ProfileId: &client.ProfileId,
		Password:  &client.Password,
		Values:    &client.Values,
	}
	resp := &api.ApiUpdateMqttClientResponse{}
	return f.request("POST", "/api/mqtt-client/"+client.Id, req, cookie, resp)
}

func (f *Fixture) ListMqttClients(cookie *http.Cookie) []api.ApiMqttClient {
	resp := &api.ApiListMqttClientsResponse{}
	f.request("GET", "/api/mqtt-client", nil, cookie, resp)
	return resp.MqttClients
}

func (f *Fixture) GetMqttClient(cookie *http.Cookie, id string) *api.ApiMqttClient {
	resp := &api.ApiMqttClient{}
	f.request("GET", "/api/mqtt-client/"+id, nil, cookie, resp)
	return resp
}

func (f *Fixture) DeleteMqttClient(cookie *http.Cookie, id string) *httptest.ResponseRecorder {
	return f.request("DELETE", "/api/mqtt-client/"+id, nil, cookie, nil)
}

func (f *Fixture) ListUsers(cookie *http.Cookie) []api.ApiUser {
	resp := &api.ApiListUsersResponse{}
	f.request("GET", "/api/user", nil, cookie, resp)
	return resp.Users
}

func (f *Fixture) CreateUser(email string) (cookie *http.Cookie, signinSecret string) {
	user, signinSecret, _ := f.Auth.CreateUser(email, email, false, nil)
	session, _ := f.Auth.CreateSession(user.Id, "user-agent", "remote-addr")
	cookie = f.Session.CreateSessionCookie(session)
	return
}

func (f *Fixture) CreateAdmin(email string) (cookie *http.Cookie, signinSecret string) {
	user, signinSecret, _ := f.Auth.CreateUser(email, email, true, nil)
	session, _ := f.Auth.CreateSession(user.Id, "user-agent", "remote-addr")
	cookie = f.Session.CreateSessionCookie(session)
	return
}

func (f *Fixture) CreateUserGetId(email string) (cookie *http.Cookie, userId string) {
	user, _, _ := f.Auth.CreateUser(email, email, false, nil)
	session, _ := f.Auth.CreateSession(user.Id, "user-agent", "remote-addr")
	cookie = f.Session.CreateSessionCookie(session)
	userId = user.Id
	return
}

func (f *Fixture) CreateAdminGetId(email string) (cookie *http.Cookie, userId string) {
	user, _, _ := f.Auth.CreateUser(email, email, true, nil)
	session, _ := f.Auth.CreateSession(user.Id, "user-agent", "remote-addr")
	cookie = f.Session.CreateSessionCookie(session)
	userId = user.Id
	return
}

func (f *Fixture) StartEnroll(cookie *http.Cookie) (*api.ApiStartEnrollResponse, error) {
	req := &api.ApiStartEnrollRequest{}
	resp := &api.ApiStartEnrollResponse{}
	rr := f.request("POST", "/api/enroll/start", req, cookie, resp)
	if rr.Code != http.StatusOK {
		return resp, fmt.Errorf("request to /api/enroll/start failed with status %d: %s", rr.Code, rr.Body.String())
	}
	if resp.Error != nil {
		return resp, fmt.Errorf("start enroll failed: %+v", resp.Error)
	}
	return resp, nil
}

func (f *Fixture) SigninFromConfirmedPollId(pollId string) *http.Cookie {
	req := &api.ApiPollSigninPinRequest{
		Id: pollId,
	}
	resp := &api.ApiPollSigninPinResponse{}
	f.request("POST", "/api/signin/pin/poll", req, nil, resp)
	cs := resp.Success.Cookie
	cookies := (&http.Response{Header: http.Header{"Set-Cookie": {cs}}}).Cookies()
	return cookies[0]
}

func (f *Fixture) request(method, url string, req interface{}, cookie *http.Cookie, res interface{}) *httptest.ResponseRecorder {
	buf := bytes.Buffer{}
	if req != nil {
		_ = json.NewEncoder(&buf).Encode(req)
	}

	httpReq, _ := http.NewRequest(method, url, &buf)

	rr := httptest.NewRecorder()
	httpReq.Host = "test.example.com"
	if cookie != nil {
		httpReq.AddCookie(cookie)
	}
	f.router.ServeHTTP(rr, httpReq)
	if rr.Code == http.StatusOK && res != nil {
		_ = json.NewDecoder(rr.Body).Decode(res)
	}
	return rr
}

func (f *Fixture) InitializeEnroll(cookie *http.Cookie, pin string) (*api.ApiEnrollRequest, error) {
	return &api.ApiEnrollRequest{}, nil
}

func (f *Fixture) FinishEnroll(cookie *http.Cookie, token string, attestationResponse *api.ApiAuthenticatorAttestationResponse) (*api.ApiFinishEnrollResponse, error) {
	jsonReq := &api.ApiFinishEnrollRequest{
		Token:               token,
		AttestationResponse: *attestationResponse,
	}
	resp := &api.ApiFinishEnrollResponse{}
	rr := f.request("POST", "/api/enroll/finish", jsonReq, cookie, resp)
	if rr.Code != http.StatusOK {
		return nil, fmt.Errorf("request to /api/enroll/finish failed with status %d: %s", rr.Code, rr.Body.String())
	}
	if resp.Error != nil {
		// This is a bit of a hack, but it's the easiest way to get a descriptive error
		// without creating a custom error type.
		return nil, fmt.Errorf("enroll finish failed: %s", rr.Body.String())
	}
	return resp, nil
}

func (f *Fixture) createSshKey(cookie *http.Cookie, name string) *api.ApiCreateSshKeyResponse {
	req := &api.ApiCreateSshKeyRequest{
		Name: name,
	}
	resp := &api.ApiCreateSshKeyResponse{}
	f.request("POST", "/api/ssh-key", req, cookie, resp)
	return resp
}

func (f *Fixture) proposeSshKey(keyID, secret, sshPubKey string) (string, error) {
	req := &api.ApiProposeSshKeyRequest{
		KeySecret: secret,
		PublicKey: sshPubKey,
	}
	res := &api.ApiProposeSshKeyResponse{}
	f.request("POST", "/api/ssh-key/"+keyID, req, nil, res)
	return res.ConfirmUrl, nil
}

func (f *Fixture) requestConfirmSshKey(cookie *http.Cookie, keyID string) *api.ApiGetConfirmSshKeyResponse {
	resp := &api.ApiGetConfirmSshKeyResponse{}
	f.request("GET", "/api/ssh-key/"+keyID+"/confirm", nil, cookie, resp)
	return resp
}

func (f *Fixture) confirmSshKey(cookie *http.Cookie, keyID, token string, credential *api.ApiAssertionCredential) *api.ApiPostConfirmSshKeyResponse {
	req := &api.ApiPostConfirmSshKeyRequest{
		Token:      token,
		Credential: *credential,
	}
	resp := &api.ApiPostConfirmSshKeyResponse{}
	f.request("POST", "/api/ssh-key/"+keyID+"/confirm", req, cookie, resp)
	return resp
}

func (f *Fixture) signinEmail(t *testing.T, email string) *api.ApiSigninEmailResponse {
	t.Helper()
	req := &api.ApiSignInEmailRequest{
		Email: email,
	}
	resp := &api.ApiSigninEmailResponse{}
	rr := f.request("POST", "/api/signin/email", req, nil, resp)
	if rr.Code != http.StatusOK {
		t.Fatalf("request to /api/signin/email failed with status %d: %s", rr.Code, rr.Body.String())
	}
	if resp.Success == nil {
		t.Fatalf("expected signinEmail to be successful, but it was not. response: %s", rr.Body.String())
	}
	return resp
}

func (f *Fixture) GenerateCredential(request *api.ApiEnrollRequest) (virtualwebauthn.Credential, *api.ApiAuthenticatorAttestationResponse) {

	authenticator := virtualwebauthn.NewAuthenticator()

	challenge, _ := base64.RawURLEncoding.DecodeString(request.Options.Challenge)

	options := virtualwebauthn.AttestationOptions{
		Challenge:          challenge,
		ExcludeCredentials: []string{},
		RelyingPartyID:     request.Options.RP.ID,
		RelyingPartyName:   request.Options.RP.Name,
		UserID:             request.Options.User.ID,
		UserName:           request.Options.User.Name,
		UserDisplayName:    request.Options.User.DisplayName,
	}
	cred := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

	rp := virtualwebauthn.RelyingParty{
		ID:     request.Options.RP.ID,
		Name:   request.Options.RP.Name,
		Origin: "https://" + request.Options.RP.ID}

	att := virtualwebauthn.CreateAttestationResponse(rp, authenticator, cred, options)

	type attestationResponse struct {
		AttestationObject string `json:"attestationObject"`
		ClientDataJSON    string `json:"clientDataJSON"`
	}

	type attestationResult struct {
		Type     string              `json:"type"`
		ID       string              `json:"id"`
		RawID    string              `json:"rawId"`
		Response attestationResponse `json:"response"`
	}
	var result attestationResult
	_ = json.Unmarshal([]byte(att), &result)

	return cred, &api.ApiAuthenticatorAttestationResponse{
		ID:                result.ID,
		AttestationObject: result.Response.AttestationObject,
		ClientDataJSON:    result.Response.ClientDataJSON,
		Transports:        []string{"internal"},
	}
}

func (f *Fixture) SignAssertionRequest(req *api.ApiAssertionRequest, userHandleB64 string, cred *virtualwebauthn.Credential) *api.ApiAssertionCredential {
	authenticator := virtualwebauthn.NewAuthenticator()
	decoded, _ := base64.RawURLEncoding.DecodeString(userHandleB64)
	authenticator.Options.UserHandle = decoded
	authenticator.AddCredential(*cred)

	challenge, _ := base64.RawURLEncoding.DecodeString(req.Challenge)

	ao := virtualwebauthn.AssertionOptions{
		Challenge: challenge,
		AllowCredentials: lo.Map(req.AllowCredentials, func(c api.ApiPublicKeyCredentialDescriptor, _ int) string {
			return c.ID
		}),
		RelyingPartyID: req.RPID,
	}
	rp := virtualwebauthn.RelyingParty{Name: req.RPID, ID: req.RPID, Origin: "https://" + req.RPID}

	ar := virtualwebauthn.CreateAssertionResponse(rp, authenticator, *cred, ao)

	type assertionResponse struct {
		AuthenticatorData string `json:"authenticatorData"`
		ClientDataJSON    string `json:"clientDataJSON"`
		Signature         string `json:"signature"`
		UserHandle        string `json:"userHandle,omitempty"`
	}

	type assertionResult struct {
		Type     string            `json:"type"`
		ID       string            `json:"id"`
		RawID    string            `json:"rawId"`
		Response assertionResponse `json:"response"`
	}
	var result assertionResult
	_ = json.Unmarshal([]byte(ar), &result)

	return &api.ApiAssertionCredential{
		ID: result.ID,
		Response: api.ApiAuthenticatorAssertionResponse{
			AuthenticatorData: result.Response.AuthenticatorData,
			ClientDataJSON:    result.Response.ClientDataJSON,
			Signature:         result.Response.Signature,
			UserHandle:        result.Response.UserHandle,
		},
	}
}
