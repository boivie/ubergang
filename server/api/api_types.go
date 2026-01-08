package api

// webauthn

type ApiAssertionRequest struct {
	Challenge        string                             `json:"challenge"`
	Timeout          int                                `json:"timeout"`
	RPID             string                             `json:"rpId"`
	AllowCredentials []ApiPublicKeyCredentialDescriptor `json:"allowCredentials"`
	UserVerification string                             `json:"userVerification"`
}

// Mapping https://www.w3.org/TR/webauthn-2/#iface-authenticatorattestationresponse
type ApiAuthenticatorAttestationResponse struct {
	ID                string   `json:"id"`
	AttestationObject string   `json:"attestationObject"`
	ClientDataJSON    string   `json:"clientDataJson"`
	Transports        []string `json:"transports"`
}

// https://www.w3.org/TR/webauthn-2/#authenticatorassertionresponse
type ApiAuthenticatorAssertionResponse struct {
	AuthenticatorData string `json:"authenticatorData"`
	ClientDataJSON    string `json:"clientDataJson"`
	Signature         string `json:"signature"`
	UserHandle        string `json:"userHandle"`
	Type              string `json:"type"`
}

type ApiAssertionCredential struct {
	ID                      string                            `json:"id"`
	AuthenticatorAttachment string                            `json:"authenticatorAttachment"`
	Response                ApiAuthenticatorAssertionResponse `json:"response"`
}

// https://www.w3.org/TR/webauthn-2/#dictdef-publickeycredentialuserentity
type ApiEnrollUser struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// https://www.w3.org/TR/webauthn-2/#dictdef-publickeycredentialrpentity
type ApiEnrollRP struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type ApiPublicKeyCredentialParameters struct {
	Type      string `json:"type"`
	Algorithm int    `json:"alg"`
}

type ApiAuthenticatorSelection struct {
	AuthenticatorAttachment string `json:"authenticatorAttachment,omitempty"`
	RequireResidentKey      *bool  `json:"requireResidentKey,omitempty"`
	ResidentKey             string `json:"residentKey,omitempty"`
	UserVerification        string `json:"userVerification,omitempty"`
}

type ApiPublicKeyCredentialDescriptor struct {
	Type       string   `json:"type"`
	ID         string   `json:"id"`
	Transports []string `json:"transports"`
}

// https://www.w3.org/TR/webauthn-2/#dictionary-makecredentialoptions
type ApiPublicKeyCredentialCreationOptions struct {
	RP                     ApiEnrollRP                        `json:"rp"`
	User                   ApiEnrollUser                      `json:"user"`
	Challenge              string                             `json:"challenge"`
	Parameters             []ApiPublicKeyCredentialParameters `json:"pubKeyCredParams"`
	Timeout                int                                `json:"timeout"`
	Attestation            string                             `json:"attestation"`
	ExcludeCredentials     []ApiPublicKeyCredentialDescriptor `json:"excludeCredentials"`
	AuthenticatorSelection ApiAuthenticatorSelection          `json:"authenticatorSelection,omitempty"`
}

type ApiEnrollRequest struct {
	Token   string                                `json:"token"`
	Options ApiPublicKeyCredentialCreationOptions `json:"options"`
}

// enroll_finish
type ApiFinishEnrollRequest struct {
	Token               string                              `json:"token"`
	AttestationResponse ApiAuthenticatorAttestationResponse `json:"attestationResponse"`
}

type ApiFinishEnrollError struct {
	InvalidEnrollment bool `json:"invalidEnrollment"`
}

type ApiFinishEnrollResponse struct {
	Credential *ApiCredential        `json:"credential,omitempty"`
	Error      *ApiFinishEnrollError `json:"error,omitempty"`
}

// enroll_start

type ApiStartEnrollRequest struct {
}

type ApiEnrollStartError struct {
}

type ApiStartEnrollResponse struct {
	Error         *ApiEnrollStartError `json:"error,omitempty"`
	EnrollRequest *ApiEnrollRequest    `json:"enrollRequest,omitempty"`
}

// credential_update
type ApiUpdateCredentialRequest struct {
	Name *string `json:"name"`
}

type ApiUpdateCredentialResponse struct {
}

// backend_update
type ApiBackendHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ApiBackend struct {
	Fqdn        string             `json:"fqdn"`
	UpstreamUrl string             `json:"upstreamUrl"`
	Headers     []ApiBackendHeader `json:"headers"`
	CreatedAt   string             `json:"createdAt"`
	UpdatedAt   string             `json:"updatedAt"`
	// Can be NORMAL or PUBLIC.
	AccessLevel string `json:"accessLevel"`
	JsScript    string `json:"jsScript"`
}

type ApiUpdateBackendRequest struct {
	UpstreamUrl *string             `json:"upstreamUrl"`
	Headers     *[]ApiBackendHeader `json:"headers"`
	AccessLevel *string             `json:"accessLevel"`
	JsScript    string              `json:"jsScript"`
}

type ApiUpdateBackendResponse struct {
}

// backend_list

type ApiListBackendsResponse struct {
	Backends []ApiBackend `json:"backends"`
}

// mqtt_profile
type ApiMqttProfile struct {
	Id             string   `json:"id"`
	AllowPublish   []string `json:"allow_publish"`
	AllowSubscribe []string `json:"allow_subscribe"`
}

type ApiUpdateMqttProfileRequest struct {
	AllowPublish   *[]string `json:"allow_publish"`
	AllowSubscribe *[]string `json:"allow_subscribe"`
}

type ApiUpdateMqttProfileResponse struct {
}

type ApiListMqttProfilesResponse struct {
	MqttProfiles []ApiMqttProfile `json:"mqtt_profiles"`
}

// mqtt_client
type ApiMqttConnected struct {
	RemoteAddr     string `json:"remoteAddr"`
	ConnectedAt    string `json:"connectedAt"`
	ConnectionType string `json:"connectionType"`
}

type ApiMqttDisconnected struct {
	RemoteAddr     string `json:"remoteAddr"`
	DisconnectedAt string `json:"disconnectedAt"`
}

type ApiMqttClient struct {
	Id           string               `json:"id"`
	ProfileId    string               `json:"profile_id"`
	Password     string               `json:"password,omitempty"`
	Values       map[string]string    `json:"values"`
	Connected    *ApiMqttConnected    `json:"connected,omitempty"`
	Disconnected *ApiMqttDisconnected `json:"disconnected,omitempty"`
}

type ApiUpdateMqttClientRequest struct {
	Id        *string            `json:"id"`
	ProfileId *string            `json:"profile_id"`
	Password  *string            `json:"password"`
	Values    *map[string]string `json:"values"`
}

type ApiUpdateMqttClientResponse struct {
}

type ApiListMqttClientsResponse struct {
	MqttClients []ApiMqttClient `json:"mqtt_clients"`
}

// user_list

type ApiListUsersResponse struct {
	Users []ApiUser `json:"users"`
}

// signin_confirm_finish

type ApiConfirmSigninPinRequest struct {
	Token      string                 `json:"token"`
	Credential ApiAssertionCredential `json:"credential"`
}

type ApiConfirmSigninPinError struct {
	InvalidEnrollment bool `json:"invalidEnrollment,omitempty"`
}

type ApiConfirmSigninPinResponse struct {
	Error *ApiConfirmSigninPinError `json:"error,omitempty"`
}

// signin_confirm_start

type ApiQuerySigninPinRequest struct {
	Pin string `json:"pin"`
}

type ApiQuerySigninPinError struct {
	InvalidPin         bool `json:"invalidPin,omitempty"`
	InvalidCredentials bool `json:"invalidCredentials,omitempty"`
}

type ApiQuerySigninPinResponse struct {
	Error              *ApiQuerySigninPinError `json:"error,omitempty"`
	ID                 string                  `json:"id,omitempty"`
	Pin                string                  `json:"pin,omitempty"`
	RequestorUserAgent string                  `json:"requestor_user_agent"`
	RequestorIP        string                  `json:"requestor_ip"`
	Token              string                  `json:"token,omitempty"`
	Confirmed          bool                    `json:"confirmed"`
	AssertionRequest   *ApiAssertionRequest    `json:"assertionRequest,omitempty"`
}

// signin_email

type ApiSignInEmailRequest struct {
	Email    string `json:"email"`
	Redirect string `json:"redirect"`
}

type ApiSignInEmailSuccess struct {
	Token            string              `json:"token"`
	AssertionRequest ApiAssertionRequest `json:"assertionRequest"`
}

type ApiSignInEmailError struct {
	WrongEmail    bool `json:"wrong_email,omitempty"`
	NoCredentials bool `json:"no_credentials,omitempty"`
	InternalError bool `json:"internal_error,omitempty"`
}

type ApiSigninEmailResponse struct {
	Error   *ApiSignInEmailError   `json:"error,omitempty"`
	Success *ApiSignInEmailSuccess `json:"success,omitempty"`
}

// signin_poll_login_request

type ApiPollSigninPinRequest struct {
	Id       string `json:"id"`
	Redirect string `json:"redirect"`
}

type ApiPollSigninPinError struct {
	InternalError bool `json:"internalError,omitempty"`
	InvalidToken  bool `json:"invalidToken,omitempty"`
	Expired       bool `json:"expired,omitempty"`
}

type ApiPollSigningPinSuccess struct {
	Cookie   string `json:"cookie"`
	Redirect string `json:"redirect"`
}

type ApiSignInPollPending struct {
	Pin        string `json:"pin"`
	ConfirmUrl string `json:"confirm_url"`
	QrCodeUrl  string `json:"qr_code_url"`
}

type ApiPollSigninPinResponse struct {
	Error   *ApiPollSigninPinError    `json:"error,omitempty"`
	Pending *ApiSignInPollPending     `json:"pending,omitempty"`
	Success *ApiPollSigningPinSuccess `json:"success,omitempty"`
}

// signin_request_pin

type ApiRequestSigninPinRequest struct {
	Email     string `json:"email"`
	UserAgent string `json:"userAgent"`
}

type ApiRequestSigninPinError struct {
	InvalidEmail bool `json:"invalidEmail"`
}

type ApiRequestSigninPinResponse struct {
	Error *ApiRequestSigninPinError `json:"error,omitempty"`
	ID    string                    `json:"id,omitempty"`
}

// signin_start

type ApiStartSigninResponse struct {
	Token            string              `json:"token"`
	AssertionRequest ApiAssertionRequest `json:"assertionRequest,omitempty"`
}

// signin_webauthn

type ApiSignInWebauthnRequest struct {
	Token      string                 `json:"token"`
	Credential ApiAssertionCredential `json:"credential"`
	Redirect   string                 `json:"redirect"`
}

type ApiSignInWebauthnError struct {
	InternalError     bool `json:"internalError,omitempty"`
	InvalidCredential bool `json:"invalidCredential,omitempty"`
}

type ApiSignInWebauthnSuccess struct {
	Cookie   string `json:"cookie"`
	Redirect string `json:"redirect"`
}

type ApiSignInWebauthResponse struct {
	Error   *ApiSignInWebauthnError   `json:"error,omitempty"`
	Success *ApiSignInWebauthnSuccess `json:"success,omitempty"`
}

// ssh_key_confirm

type ApiPostConfirmSshKeyRequest struct {
	Token      string                 `json:"token"`
	Credential ApiAssertionCredential `json:"credential"`
}

type ApiPostConfirmSshKeyResult struct {
	ExpiresAt string `json:"expiresAt"`
}

type ApiPostConfirmSshKeyError struct {
	FailedAuthentication bool `json:"failedAuthentication,omitempty"`
}

type ApiPostConfirmSshKeyResponse struct {
	Error  *ApiPostConfirmSshKeyError  `json:"error,omitempty"`
	Result *ApiPostConfirmSshKeyResult `json:"result,omitempty"`
}

// ssh_key_create

type ApiCreateSshKeyRequest struct {
	Name string `json:"name"`
}

type ApiCreateSshKeyResponse struct {
	KeyID string `json:"keyId"`
}

// ssh_key_get

type ApiSSHKey struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	CreatedAt         string `json:"createdAt"`
	ExpiresAt         string `json:"expiresAt"`
	Sha256Fingerprint string `json:"sha256Fingerprint"`
}

// ssh_key_request_confirm

type ApiGetConfirmSshKeyError struct {
	InvalidKey bool `json:"invalidKey"`
}

type ApiGetConfirmSshKeyAuthenticate struct {
	KeyName          string              `json:"keyName"`
	Token            string              `json:"token"`
	AssertionRequest ApiAssertionRequest `json:"assertionRequest"`
}

type ApiGetConfirmSshKeyResponse struct {
	Error        *ApiGetConfirmSshKeyError        `json:"error,omitempty"`
	Authenticate *ApiGetConfirmSshKeyAuthenticate `json:"authenticate,omitempty"`
}

// ssh_key_update

type ApiProposeSshKeyRequest struct {
	KeySecret string `json:"keySecret"`
	PublicKey string `json:"publicKey"`
}

type ApiProposeSshKeyResponse struct {
	ConfirmUrl string `json:"confirmUrl"`
}

// api_user

type ApiSession struct {
	ID         string `json:"id"`
	UserAgent  string `json:"userAgent"`
	RemoteAddr string `json:"remoteAddr"`
	CreatedAt  string `json:"createdAt"`
	AccessedAt string `json:"accessedAt"`
}

type ApiCredential struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	CreatedAt  string   `json:"createdAt"`
	CreatedBy  string   `json:"created_by_session_id"`
	Transports []string `json:"transports"`
	LastUsedAt string   `json:"lastUsedAt"`
	UsedBy     []string `json:"used_by_session_ids"`
	Aaguid     string   `json:"aaguid"`
}

type ApiUser struct {
	ID             string          `json:"id"`
	Email          string          `json:"email"`
	DisplayName    string          `json:"displayName"`
	AllowedHosts   []string        `json:"allowedHosts"`
	IsAdmin        bool            `json:"isAdmin"`
	Credentials    []ApiCredential `json:"credentials"`
	Sessions       []ApiSession    `json:"sessions"`
	CurrentSession *ApiSession     `json:"currentSession"`
	SSHKeys        []ApiSSHKey     `json:"sshKeys"`
}

// user_create

type ApiCreateUserRequest struct {
	Email string `json:"email"`
}

type ApiCreateUserResponse struct {
	ID string `json:"id"`
}

// user_update

type ApiUpdateUserRequest struct {
	Email        *string   `json:"email,omitempty"`
	DisplayName  *string   `json:"displayName,omitempty"`
	Admin        *bool     `json:"admin,omitempty"`
	AllowedHosts *[]string `json:"allowedHosts,omitempty"`
}

type ApiUpdateUserResponse struct {
}

// user_recover

type ApiUserRecoverResponse struct {
	RecoveryUrl string `json:"recoveryUrl"`
}

// testing_setup

type ApiTestingSetupResponse struct {
	SigninUrl string `json:"signinUrl"`
}

// bootstrap

type ApiBootstrapConfigureRequest struct {
	Email    string `json:"email"`
	SiteFqdn string `json:"siteFqdn"`
}

type ApiBootstrapConfigureResponse struct {
	AdminFqdn string `json:"admin_fqdn"`
}

type ApiBootstrapStatusResponse struct {
	IsConfigured bool `json:"isConfigured"`
}
