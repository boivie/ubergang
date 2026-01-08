export interface ApiAssertionRequest {
  challenge: string;
  timeout: number;
  rpId: string;
  allowCredentials: ApiPublicKeyCredentialDescriptor[];
  userVerification: string;
}

export interface ApiAuthenticatorAttestationResponse {
  id: string;
  attestationObject: string;
  clientDataJson: string;
  transports: string[];
}

export interface ApiAuthenticatorAssertionResponse {
  authenticatorData: string;
  clientDataJson: string;
  signature: string;
  userHandle: string;
  type: string;
}

export interface ApiAssertionCredential {
  id: string;
  authenticatorAttachment: string;
  response: ApiAuthenticatorAssertionResponse;
}

export interface ApiEnrollUser {
  name: string;
  id: string;
  displayName: string;
}

export interface ApiEnrollRP {
  name: string;
  id: string;
}

export interface ApiPublicKeyCredentialParameters {
  type: string;
  alg: number;
}

export interface ApiAuthenticatorSelection {
  authenticatorAttachment?: string;
  requireResidentKey?: boolean;
  residentKey?: string;
  userVerification?: string;
}

export interface ApiPublicKeyCredentialDescriptor {
  type: string;
  id: string;
  transports: string[];
}

export interface ApiPublicKeyCredentialCreationOptions {
  rp: ApiEnrollRP;
  user: ApiEnrollUser;
  challenge: string;
  pubKeyCredParams: ApiPublicKeyCredentialParameters[];
  timeout: number;
  attestation: string;
  excludeCredentials: ApiPublicKeyCredentialDescriptor[];
  authenticatorSelection?: ApiAuthenticatorSelection;
}

export interface ApiEnrollRequest {
  token: string;
  options: ApiPublicKeyCredentialCreationOptions;
}

export interface ApiFinishEnrollRequest {
  token: string;
  attestationResponse: ApiAuthenticatorAttestationResponse;
}

export interface ApiFinishEnrollError {
  invalidEnrollment: boolean;
}

export interface ApiFinishEnrollResponse {
  credential?: ApiCredential;
  error?: ApiFinishEnrollError;
}

export interface ApiStartEnrollRequest {}

export interface ApiEnrollStartError {}

export interface ApiStartEnrollResponse {
  error?: ApiEnrollStartError;
  enrollRequest?: ApiEnrollRequest;
}

export interface ApiUpdateCredentialRequest {
  name?: string;
}

export interface ApiUpdateCredentialResponse {}

export interface ApiBackendHeader {
  name: string;
  value: string;
}

export interface ApiBackend {
  fqdn: string;
  upstreamUrl: string;
  headers: ApiBackendHeader[];
  createdAt: string;
  updatedAt: string;
  accessLevel: string;
  jsScript: string;
}

export interface ApiUpdateBackendRequest {
  upstreamUrl?: string;
  headers?: ApiBackendHeader[];
  accessLevel?: string;
  jsScript?: string;
}

export interface ApiUpdateBackendResponse {}

export interface ApiListBackendsResponse {
  backends: ApiBackend[];
}

export interface ApiMqttProfile {
  id: string;
  allow_publish: string[];
  allow_subscribe: string[];
}

export interface ApiUpdateMqttProfileRequest {
  allow_publish?: string[];
  allow_subscribe?: string[];
}

export interface ApiUpdateMqttProfileResponse {}

export interface ApiListMqttProfilesResponse {
  mqtt_profiles: ApiMqttProfile[];
}

export interface ApiMqttConnected {
  remoteAddr: string;
  connectedAt: string;
  connectionType: string;
}

export interface ApiMqttDisconnected {
  remoteAddr: string;
  disconnectedAt: string;
}

export interface ApiMqttClient {
  id: string;
  profile_id: string;
  password?: string;
  values: { [key: string]: string };
  connected?: ApiMqttConnected;
  disconnected?: ApiMqttDisconnected;
}

export interface ApiUpdateMqttClientRequest {
  id?: string;
  profile_id?: string;
  password?: string;
  values?: { [key: string]: string };
}

export interface ApiUpdateMqttClientResponse {}

export interface ApiListMqttClientsResponse {
  mqtt_clients: ApiMqttClient[];
}

export interface ApiListUsersResponse {
  users: ApiUser[];
}

export interface ApiConfirmSigninPinRequest {
  token: string;
  credential: ApiAssertionCredential;
}

export interface ApiConfirmSigninPinError {
  invalidEnrollment?: boolean;
}

export interface ApiConfirmSigninPinResponse {
  error?: ApiConfirmSigninPinError;
}

export interface ApiQuerySigninPinRequest {
  pin: string;
}

export interface ApiQuerySigninPinError {
  invalidPin?: boolean;
  invalidCredentials?: boolean;
}

export interface ApiQuerySigninPinResponse {
  error?: ApiQuerySigninPinError;
  id?: string;
  pin?: string;
  requestor_user_agent: string;
  requestor_ip: string;
  token?: string;
  confirmed: boolean;
  assertionRequest?: ApiAssertionRequest;
}

export interface ApiSignInEmailRequest {
  email: string;
  redirect: string;
}

export interface ApiSignInEmailSuccess {
  token: string;
  assertionRequest: ApiAssertionRequest;
}

export interface ApiSignInEmailError {
  wrong_email?: boolean;
  no_credentials?: boolean;
  internal_error?: boolean;
}

export interface ApiSigninEmailResponse {
  error?: ApiSignInEmailError;
  success?: ApiSignInEmailSuccess;
}

export interface ApiPollSigninPinRequest {
  id: string;
  redirect: string;
}

export interface ApiPollSigninPinError {
  internalError?: boolean;
  invalidToken?: boolean;
  expired?: boolean;
}

export interface ApiPollSigningPinSuccess {
  cookie: string;
  redirect: string;
}

export interface ApiSignInPollPending {
  pin: string;
  confirm_url: string;
  qr_code_url: string;
}

export interface ApiPollSigninPinResponse {
  error?: ApiPollSigninPinError;
  pending?: ApiSignInPollPending;
  success?: ApiPollSigningPinSuccess;
}

export interface ApiRequestSigninPinRequest {
  email: string;
  userAgent: string;
}

export interface ApiRequestSigninPinError {
  invalidEmail: boolean;
}

export interface ApiRequestSigninPinResponse {
  error?: ApiRequestSigninPinError;
  id?: string;
}

export interface ApiStartSigninResponse {
  token: string;
  assertionRequest?: ApiAssertionRequest;
}

export interface ApiSignInWebauthnRequest {
  token: string;
  credential: ApiAssertionCredential;
  redirect: string;
}

export interface ApiSignInWebauthnError {
  internalError?: boolean;
  invalidCredential?: boolean;
}

export interface ApiSignInWebauthnSuccess {
  cookie: string;
  redirect: string;
}

export interface ApiSignInWebauthResponse {
  error?: ApiSignInWebauthnError;
  success?: ApiSignInWebauthnSuccess;
}

export interface ApiPostConfirmSshKeyRequest {
  token: string;
  credential: ApiAssertionCredential;
}

export interface ApiPostConfirmSshKeyResult {
  expiresAt: string;
}

export interface ApiPostConfirmSshKeyError {
  failedAuthentication?: boolean;
}

export interface ApiPostConfirmSshKeyResponse {
  error?: ApiPostConfirmSshKeyError;
  result?: ApiPostConfirmSshKeyResult;
}

export interface ApiCreateSshKeyRequest {
  name: string;
}

export interface ApiCreateSshKeyResponse {
  keyId: string;
}

export interface ApiSSHKey {
  id: string;
  name: string;
  createdAt: string;
  expiresAt: string;
  sha256Fingerprint: string;
}

export interface ApiGetConfirmSshKeyError {
  invalidKey: boolean;
}

export interface ApiGetConfirmSshKeyAuthenticate {
  keyName: string;
  token: string;
  assertionRequest: ApiAssertionRequest;
}

export interface ApiGetConfirmSshKeyResponse {
  error?: ApiGetConfirmSshKeyError;
  authenticate?: ApiGetConfirmSshKeyAuthenticate;
}

export interface ApiProposeSshKeyRequest {
  keySecret: string;
  publicKey: string;
}

export interface ApiProposeSshKeyResponse {
  confirmUrl: string;
}

export interface ApiSession {
  id: string;
  userAgent: string;
  remoteAddr: string;
  createdAt: string;
  accessedAt: string;
}

export interface ApiCredential {
  id: string;
  name: string;
  type: string;
  createdAt: string;
  created_by_session_id: string;
  transports: string[];
  lastUsedAt: string;
  used_by_session_ids: string[];
  aaguid: string;
}

export interface ApiUser {
  id: string;
  email: string;
  displayName: string;
  allowedHosts: string[];
  isAdmin: boolean;
  credentials: ApiCredential[];
  sessions: ApiSession[];
  currentSession?: ApiSession;
  sshKeys: ApiSSHKey[];
}

export interface ApiCreateUserRequest {
  email: string;
}

export interface ApiCreateUserResponse {
  id: string;
}

export interface ApiUpdateUserRequest {
  email?: string;
  displayName?: string;
  admin?: boolean;
  allowedHosts?: string[];
}

export interface ApiUpdateUserResponse {}

export interface ApiUserRecoverResponse {
  recoveryUrl: string;
}

export interface ApiTestingSetupResponse {
  signinUrl: string;
}

export interface ApiBootstrapConfigureRequest {
  email: string;
  siteFqdn: string;
}

export interface ApiBootstrapConfigureResponse {
  admin_fqdn: string;
}

export interface ApiBootstrapStatusResponse {
  isConfigured: boolean;
}
