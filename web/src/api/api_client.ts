import { createContext, useContext } from "react";
import {
  ApiBackend,
  ApiConfirmSigninPinRequest,
  ApiConfirmSigninPinResponse,
  ApiCreateUserRequest,
  ApiCreateUserResponse,
  ApiFinishEnrollRequest,
  ApiFinishEnrollResponse,
  ApiGetConfirmSshKeyResponse,
  ApiListBackendsResponse,
  ApiListMqttClientsResponse,
  ApiListMqttProfilesResponse,
  ApiListUsersResponse,
  ApiMqttClient,
  ApiMqttProfile,
  ApiPollSigninPinRequest,
  ApiPollSigninPinResponse,
  ApiPostConfirmSshKeyRequest,
  ApiPostConfirmSshKeyResponse,
  ApiQuerySigninPinRequest,
  ApiQuerySigninPinResponse,
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
} from "./api_types";

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
