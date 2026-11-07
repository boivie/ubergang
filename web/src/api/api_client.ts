import { createContext, useContext } from "react";
import {
  ApiBackend,
  ApiApproveKioskSigninRequest,
  ApiApproveKioskSigninResponse,
  ApiConfirmSigninPinRequest,
  ApiConfirmSigninPinResponse,
  ApiCreateApiKeyRequest,
  ApiCreateApiKeyResponse,
  ApiCreateUserRequest,
  ApiCreateUserResponse,
  ApiFinishEnrollRequest,
  ApiFinishEnrollResponse,
  ApiGetConfirmSshKeyResponse,
  ApiListApiKeysResponse,
  ApiListBackendsResponse,
  ApiListMqttClientsResponse,
  ApiListMqttProfilesResponse,
  ApiListUsersResponse,
  ApiMqttClient,
  ApiMqttProfile,
  ApiPollKioskSigninRequest,
  ApiPollKioskSigninResponse,
  ApiPollSigninPinRequest,
  ApiPollSigninPinResponse,
  ApiPostConfirmSshKeyRequest,
  ApiPostConfirmSshKeyResponse,
  ApiQueryKioskSigninRequest,
  ApiQueryKioskSigninResponse,
  ApiQuerySigninPinRequest,
  ApiQuerySigninPinResponse,
  ApiRequestKioskSigninRequest,
  ApiRequestKioskSigninResponse,
  ApiRequestSigninPinRequest,
  ApiRequestSigninPinResponse,
  ApiSigninEmailResponse,
  ApiSignInWebauthnRequest,
  ApiSignInWebauthResponse,
  ApiStartEnrollResponse,
  ApiStartSigninResponse,
  ApiUpdateBackendRequest,
  ApiUpdateBackendResponse,
  ApiUpdateCredentialRequest,
  ApiUpdateCredentialResponse,
  ApiUpdateMqttClientRequest,
  ApiUpdateMqttClientResponse,
  ApiUpdateMqttProfileRequest,
  ApiUpdateMqttProfileResponse,
  ApiUpdateUserRequest,
  ApiUpdateUserResponse,
  ApiUser,
  ApiUserRecoverResponse,
  ApiBootstrapConfigureRequest,
  ApiBootstrapConfigureResponse,
  ApiBootstrapStatusResponse,
  ApiValidateBackendResponse,
} from "./api_types";

// Thrown when an API call requires the device to be signed in, and it isn't.
export class NotSignedInError extends Error {
  constructor() {
    super("Not signed in");
    this.name = "NotSignedInError";
  }
}

async function postJson<T>(path: string, req: unknown): Promise<T> {
  const res = await fetch(path, {
    method: "post",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify(req),
  });
  if (res.status === 403) {
    throw new NotSignedInError();
  }
  if (!res.ok) {
    throw new Error(`Request to ${path} failed: ${res.statusText}`);
  }
  return res.json();
}

export interface ApiService {
  GetPasswordlessToken(): Promise<ApiStartSigninResponse>;

  SignInEmail(email: string): Promise<ApiSigninEmailResponse>;

  SignInWebauthn(
    req: ApiSignInWebauthnRequest,
  ): Promise<ApiSignInWebauthResponse>;

  GetUser(userId: string): Promise<ApiUser>;

  StartEnroll(): Promise<ApiStartEnrollResponse>;

  FinishEnroll(req: ApiFinishEnrollRequest): Promise<ApiFinishEnrollResponse>;

  RequestSigninPin(
    req: ApiRequestSigninPinRequest,
  ): Promise<ApiRequestSigninPinResponse>;

  PollLoginRequest(
    req: ApiPollSigninPinRequest,
  ): Promise<ApiPollSigninPinResponse>;

  QuerySigninPin(
    req: ApiQuerySigninPinRequest,
  ): Promise<ApiQuerySigninPinResponse>;

  ConfirmSigninPin(
    req: ApiConfirmSigninPinRequest,
  ): Promise<ApiConfirmSigninPinResponse>;

  RequestKioskSignin(
    req: ApiRequestKioskSigninRequest,
  ): Promise<ApiRequestKioskSigninResponse>;

  PollKioskSignin(
    req: ApiPollKioskSigninRequest,
  ): Promise<ApiPollKioskSigninResponse>;

  // Throws NotSignedInError if this device isn't signed in.
  QueryKioskSignin(
    req: ApiQueryKioskSigninRequest,
  ): Promise<ApiQueryKioskSigninResponse>;

  // Throws NotSignedInError if this device isn't signed in.
  ApproveKioskSignin(
    req: ApiApproveKioskSigninRequest,
  ): Promise<ApiApproveKioskSigninResponse>;

  GetConfirmSshKey(shortKeyId: string): Promise<ApiGetConfirmSshKeyResponse>;

  PostConfirmSshKey(
    shortKeyId: string,
    req: ApiPostConfirmSshKeyRequest,
  ): Promise<ApiPostConfirmSshKeyResponse>;

  ListBackends(): Promise<ApiListBackendsResponse>;

  GetBackend(fqdn: string): Promise<ApiBackend>;

  UpdateBackend(
    fqdn: string,
    req: ApiUpdateBackendRequest,
  ): Promise<ApiUpdateBackendResponse>;

  DeleteBackend(fqdn: string): Promise<void>;

  ValidateBackend(fqdn: string): Promise<ApiValidateBackendResponse>;

  DeleteCredential(id: string): Promise<void>;

  DeleteSession(id: string): Promise<void>;

  UpdateCredential(
    id: string,
    req: ApiUpdateCredentialRequest,
  ): Promise<ApiUpdateCredentialResponse>;

  ListUsers(): Promise<ApiListUsersResponse>;

  CreateUser(req: ApiCreateUserRequest): Promise<ApiCreateUserResponse>;

  UpdateUser(
    userId: string,
    req: ApiUpdateUserRequest,
  ): Promise<ApiUpdateUserResponse>;

  DeleteUser(userId: string): Promise<void>;

  RecoverUser(userId: string): Promise<ApiUserRecoverResponse>;

  ListApiKeys(): Promise<ApiListApiKeysResponse>;

  CreateApiKey(req: ApiCreateApiKeyRequest): Promise<ApiCreateApiKeyResponse>;

  DeleteApiKey(id: string): Promise<void>;

  ListMqttProfiles(): Promise<ApiListMqttProfilesResponse>;

  GetMqttProfile(id: string): Promise<ApiMqttProfile>;

  UpdateMqttProfile(
    id: string,
    req: ApiUpdateMqttProfileRequest,
  ): Promise<ApiUpdateMqttProfileResponse>;

  DeleteMqttProfile(id: string): Promise<void>;

  ListMqttClients(): Promise<ApiListMqttClientsResponse>;

  GetMqttClient(id: string): Promise<ApiMqttClient>;

  UpdateMqttClient(
    id: string,
    req: ApiUpdateMqttClientRequest,
  ): Promise<ApiUpdateMqttClientResponse>;

  DeleteMqttClient(id: string): Promise<void>;

  ImportMqtt(file: File): Promise<{
    success: boolean;
    profiles_count: number;
    clients_count: number;
  }>;

  ExportMqtt(): Promise<Blob>;

  GetBootstrapStatus(): Promise<ApiBootstrapStatusResponse>;

  BootstrapConfigure(
    req: ApiBootstrapConfigureRequest,
  ): Promise<ApiBootstrapConfigureResponse>;
}

export const realApiService: ApiService = {
  async GetPasswordlessToken(): Promise<ApiStartSigninResponse> {
    const res = await fetch(`/api/signin/start`, {
      method: "get",
      headers: {
        Accept: "application/json",
      },
    });
    return res.json();
  },

  async SignInEmail(email: string): Promise<ApiSigninEmailResponse> {
    const res = await fetch("/api/signin/email", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        email,
      }),
    });
    return res.json();
  },

  async SignInWebauthn(
    req: ApiSignInWebauthnRequest,
  ): Promise<ApiSignInWebauthResponse> {
    const res = await fetch("/api/signin/webauthn", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    return res.json();
  },

  async GetUser(userId: string): Promise<ApiUser> {
    const res = await fetch(`/api/user/${userId}`, {
      method: "get",
      headers: { Accept: "application/json" },
    });
    return res.json();
  },

  async StartEnroll(): Promise<ApiStartEnrollResponse> {
    const res = await fetch("/api/enroll/start", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({}),
    });
    return res.json();
  },

  async FinishEnroll(
    req: ApiFinishEnrollRequest,
  ): Promise<ApiFinishEnrollResponse> {
    const res = await fetch("/api/enroll/finish", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    return res.json();
  },

  async RequestSigninPin(
    req: ApiRequestSigninPinRequest,
  ): Promise<ApiRequestSigninPinResponse> {
    const res = await fetch("/api/signin/pin/request", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    return res.json();
  },

  async PollLoginRequest(
    req: ApiPollSigninPinRequest,
  ): Promise<ApiPollSigninPinResponse> {
    const res = await fetch("/api/signin/pin/poll", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    return res.json();
  },

  async QuerySigninPin(
    req: ApiQuerySigninPinRequest,
  ): Promise<ApiQuerySigninPinResponse> {
    const res = await fetch("/api/signin/pin/query", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    return res.json();
  },

  async ConfirmSigninPin(
    req: ApiConfirmSigninPinRequest,
  ): Promise<ApiConfirmSigninPinResponse> {
    const res = await fetch("/api/signin/pin/confirm", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    return res.json();
  },

  async RequestKioskSignin(
    req: ApiRequestKioskSigninRequest,
  ): Promise<ApiRequestKioskSigninResponse> {
    return postJson("/api/kiosk/request", req);
  },

  async PollKioskSignin(
    req: ApiPollKioskSigninRequest,
  ): Promise<ApiPollKioskSigninResponse> {
    return postJson("/api/kiosk/poll", req);
  },

  async QueryKioskSignin(
    req: ApiQueryKioskSigninRequest,
  ): Promise<ApiQueryKioskSigninResponse> {
    return postJson("/api/kiosk/query", req);
  },

  async ApproveKioskSignin(
    req: ApiApproveKioskSigninRequest,
  ): Promise<ApiApproveKioskSigninResponse> {
    return postJson("/api/kiosk/approve", req);
  },

  async GetConfirmSshKey(
    shortKeyId: string,
  ): Promise<ApiGetConfirmSshKeyResponse> {
    const res = await fetch(`/api/ssh-key/${shortKeyId}/confirm`, {
      method: "get",
      headers: { Accept: "application/json" },
    });
    return res.json();
  },

  async PostConfirmSshKey(
    shortKeyId: string,
    req: ApiPostConfirmSshKeyRequest,
  ): Promise<ApiPostConfirmSshKeyResponse> {
    const res = await fetch(`/api/ssh-key/${shortKeyId}/confirm`, {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    return res.json();
  },

  async ListBackends(): Promise<ApiListBackendsResponse> {
    const res = await fetch("/api/backend", {
      method: "get",
      headers: { Accept: "application/json" },
    });
    return res.json();
  },

  async GetBackend(fqdn: string): Promise<ApiBackend> {
    const res = await fetch(`/api/backend/${fqdn}`, {
      method: "get",
      headers: { Accept: "application/json" },
    });
    return res.json();
  },

  async UpdateBackend(
    fqdn: string,
    req: ApiUpdateBackendRequest,
  ): Promise<ApiUpdateBackendResponse> {
    const res = await fetch(`/api/backend/${fqdn}`, {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      const message = (await res.text()).trim();
      throw new Error(message || `Failed to update backend: ${res.statusText}`);
    }
    return res.json();
  },

  async DeleteBackend(fqdn: string): Promise<void> {
    const res = await fetch(`/api/backend/${fqdn}`, {
      method: "delete",
      headers: { Accept: "application/json" },
    });
    if (!res.ok) {
      throw new Error(`Failed to delete backend: ${res.statusText}`);
    }
  },

  async ValidateBackend(
    upstreamUrl: string,
  ): Promise<ApiValidateBackendResponse> {
    const res = await fetch(
      `/api/backend/validate?upstream_url=${encodeURIComponent(upstreamUrl)}`,
      {
        method: "post",
        headers: { Accept: "application/json" },
      },
    );
    if (!res.ok) {
      throw new Error(`Failed to validate backend: ${res.statusText}`);
    }
    return res.json();
  },

  async DeleteCredential(id: string): Promise<void> {
    const res = await fetch(`/api/credential/${id}`, {
      method: "delete",
      headers: { Accept: "application/json" },
    });
    if (!res.ok) {
      throw new Error(`Failed to delete credential: ${res.statusText}`);
    }
  },

  async DeleteSession(id: string): Promise<void> {
    const res = await fetch(`/api/session/${id}`, {
      method: "delete",
      headers: { Accept: "application/json" },
    });
    if (!res.ok) {
      throw new Error(`Failed to delete session: ${res.statusText}`);
    }
  },

  async UpdateCredential(
    id: string,
    req: ApiUpdateCredentialRequest,
  ): Promise<ApiUpdateCredentialResponse> {
    const res = await fetch(`/api/credential/${id}`, {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    return res.json();
  },

  async ListUsers(): Promise<ApiListUsersResponse> {
    const res = await fetch("/api/user", {
      method: "get",
      headers: { Accept: "application/json" },
    });
    return res.json();
  },

  async CreateUser(req: ApiCreateUserRequest): Promise<ApiCreateUserResponse> {
    const res = await fetch("/api/user", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      throw new Error(`Failed to create user: ${res.statusText}`);
    }
    return res.json();
  },

  async UpdateUser(
    userId: string,
    req: ApiUpdateUserRequest,
  ): Promise<ApiUpdateUserResponse> {
    const res = await fetch(`/api/user/${userId}`, {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      throw new Error(`Failed to update user: ${res.statusText}`);
    }
    return res.json();
  },

  async DeleteUser(userId: string): Promise<void> {
    const res = await fetch(`/api/user/${userId}`, {
      method: "delete",
      headers: { Accept: "application/json" },
    });
    if (!res.ok) {
      throw new Error(`Failed to delete user: ${res.statusText}`);
    }
  },

  async RecoverUser(userId: string): Promise<ApiUserRecoverResponse> {
    const res = await fetch(`/api/user/${userId}/recover`, {
      method: "post",
      headers: { Accept: "application/json" },
    });
    if (!res.ok) {
      throw new Error(`Failed to generate recovery URL: ${res.statusText}`);
    }
    return res.json();
  },

  async ListApiKeys(): Promise<ApiListApiKeysResponse> {
    const res = await fetch("/api/api-key", {
      method: "get",
      headers: { Accept: "application/json" },
    });
    if (!res.ok) {
      throw new Error(`Failed to list API keys: ${res.statusText}`);
    }
    return res.json();
  },

  async CreateApiKey(
    req: ApiCreateApiKeyRequest,
  ): Promise<ApiCreateApiKeyResponse> {
    const res = await fetch("/api/api-key", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      throw new Error(`Failed to create API key: ${res.statusText}`);
    }
    return res.json();
  },

  async DeleteApiKey(id: string): Promise<void> {
    const res = await fetch(`/api/api-key/${encodeURIComponent(id)}`, {
      method: "delete",
      headers: { Accept: "application/json" },
    });
    if (!res.ok) {
      throw new Error(`Failed to delete API key: ${res.statusText}`);
    }
  },

  async ListMqttProfiles(): Promise<ApiListMqttProfilesResponse> {
    const res = await fetch("/api/mqtt-profile", {
      method: "get",
      headers: { Accept: "application/json" },
    });
    return res.json();
  },

  async GetMqttProfile(id: string): Promise<ApiMqttProfile> {
    const res = await fetch(`/api/mqtt-profile/${id}`, {
      method: "get",
      headers: { Accept: "application/json" },
    });
    return res.json();
  },

  async UpdateMqttProfile(
    id: string,
    req: ApiUpdateMqttProfileRequest,
  ): Promise<ApiUpdateMqttProfileResponse> {
    const res = await fetch(`/api/mqtt-profile/${id}`, {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    return res.json();
  },

  async DeleteMqttProfile(id: string): Promise<void> {
    const res = await fetch(`/api/mqtt-profile/${id}`, {
      method: "delete",
      headers: { Accept: "application/json" },
    });
    if (!res.ok) {
      throw new Error(`Failed to delete MQTT Profile: ${res.statusText}`);
    }
  },

  async ListMqttClients(): Promise<ApiListMqttClientsResponse> {
    const res = await fetch("/api/mqtt-client", {
      method: "get",
      headers: { Accept: "application/json" },
    });
    return res.json();
  },

  async GetMqttClient(id: string): Promise<ApiMqttClient> {
    const res = await fetch(`/api/mqtt-client/${id}`, {
      method: "get",
      headers: { Accept: "application/json" },
    });
    return res.json();
  },

  async UpdateMqttClient(
    id: string,
    req: ApiUpdateMqttClientRequest,
  ): Promise<ApiUpdateMqttClientResponse> {
    const res = await fetch(`/api/mqtt-client/${id}`, {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    return res.json();
  },

  async DeleteMqttClient(id: string): Promise<void> {
    const res = await fetch(`/api/mqtt-client/${id}`, {
      method: "delete",
      headers: { Accept: "application/json" },
    });
    if (!res.ok) {
      throw new Error(`Failed to delete MQTT client: ${res.statusText}`);
    }
  },

  async ImportMqtt(file: File): Promise<{
    success: boolean;
    profiles_count: number;
    clients_count: number;
  }> {
    const res = await fetch("/api/mqtt/import", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/x-yaml",
      },
      body: file,
    });
    if (!res.ok) {
      const errorText = await res.text();
      throw new Error(`Failed to import MQTT configuration: ${errorText}`);
    }
    return res.json();
  },

  async ExportMqtt(): Promise<Blob> {
    const res = await fetch("/api/mqtt/export", {
      method: "get",
      headers: {
        Accept: "application/x-yaml",
      },
    });
    if (!res.ok) {
      throw new Error(`Failed to export MQTT configuration: ${res.statusText}`);
    }
    return res.blob();
  },

  async GetBootstrapStatus(): Promise<ApiBootstrapStatusResponse> {
    const res = await fetch("/api/bootstrap/status", {
      method: "get",
      headers: { Accept: "application/json" },
    });
    return res.json();
  },

  async BootstrapConfigure(
    req: ApiBootstrapConfigureRequest,
  ): Promise<ApiBootstrapConfigureResponse> {
    const res = await fetch("/api/bootstrap/configure", {
      method: "post",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      throw new Error(`Failed to configure: ${res.statusText}`);
    }
    return res.json();
  },
};

export const ApiServiceContext = createContext<ApiService | undefined>(
  undefined,
);

export const useApiService = () => {
  const context = useContext<ApiService | undefined>(ApiServiceContext);
  if (context === undefined) {
    throw new Error("ApiServiceContext must wrap this component");
  }
  return context;
};
