import {
  ApiAuthenticatorAttestationResponse,
  ApiAssertionCredential,
  ApiEnrollRequest,
  ApiAssertionRequest,
} from "../api/api_types";
import { base64Decode, base64Encode } from "./utils";

export interface WebauthnAssertionRequest {
  request: ApiAssertionRequest;
  onCredential: (credential: ApiAssertionCredential) => void;
  onAssertionError: (error: Error) => void;
  onNotSupported: () => void;
}

export interface WebauthnAttestationRequest {
  enroll: ApiEnrollRequest;
  onCredential: (
    attestationResponse: ApiAuthenticatorAttestationResponse,
  ) => void;
  onError: (error: Error) => void;
  onNotSupported: () => void;
}

export interface WebauthnService {
  startAssertion(request: WebauthnAssertionRequest): void;
  startConditionalAssertion(request: WebauthnAssertionRequest): void;
  startAttestation(request: WebauthnAttestationRequest): void;
}

interface WebauthnServiceWithController extends WebauthnService {
  controller?: AbortController;
}

export const realWebauthnService: WebauthnServiceWithController = {
  controller: undefined as AbortController | undefined,

  startConditionalAssertion(request: WebauthnAssertionRequest) {
    const req = request.request;
    const opts: PublicKeyCredentialRequestOptions = {
      challenge: base64Decode(req.challenge),
      rpId: req.rpId,
      timeout: req.timeout,
      userVerification: req.userVerification as UserVerificationRequirement,
      allowCredentials: req.allowCredentials.map((c) => ({
        type: c.type as PublicKeyCredentialType,
        id: base64Decode(c.id),
        transports: c.transports as AuthenticatorTransport[],
      })),
    };
    const navigatorObj = window.navigator;
    if (!navigatorObj || !navigatorObj.credentials) {
      request.onNotSupported();
      return;
    }
    this.controller = new AbortController();
    navigatorObj.credentials
      .get({
        mediation: "conditional",
        publicKey: opts,
        signal: this.controller.signal,
      })
      .then((credential: Credential | null) => {
        const assertion = credential as PublicKeyCredential & {
          response: AuthenticatorAssertionResponse;
        };
        const assertionCredential: ApiAssertionCredential = {
          id: base64Encode(assertion.rawId),
          authenticatorAttachment: assertion.authenticatorAttachment || "",
          response: {
            authenticatorData: base64Encode(
              assertion.response.authenticatorData,
            ),
            clientDataJson: base64Encode(assertion.response.clientDataJSON),
            signature: base64Encode(assertion.response.signature),
            userHandle: base64Encode(
              assertion.response.userHandle || new ArrayBuffer(0),
            ),
            type: "public-key",
          },
        };
        request.onCredential(assertionCredential);
      })
      .catch((error: Error) => {
        request.onAssertionError(error);
      });
  },
  startAssertion(request: WebauthnAssertionRequest) {
    if (this.controller) {
      this.controller.abort();
    }
    const req = request.request;
    const opts: PublicKeyCredentialRequestOptions = {
      challenge: base64Decode(req.challenge),
      rpId: req.rpId,
      timeout: req.timeout,
      userVerification: req.userVerification as UserVerificationRequirement,
      allowCredentials: req.allowCredentials.map((c) => ({
        type: c.type as PublicKeyCredentialType,
        id: base64Decode(c.id),
        transports: c.transports as AuthenticatorTransport[],
      })),
    };
    const navigatorObj = window.navigator;
    if (!navigatorObj || !navigatorObj.credentials) {
      request.onNotSupported();
      return;
    }
    navigatorObj.credentials
      .get({ publicKey: opts })
      .then((credential: Credential | null) => {
        const assertion = credential as PublicKeyCredential & {
          response: AuthenticatorAssertionResponse;
        };
        const assertionCredential: ApiAssertionCredential = {
          id: base64Encode(assertion.rawId),
          authenticatorAttachment: assertion.authenticatorAttachment || "",
          response: {
            authenticatorData: base64Encode(
              assertion.response.authenticatorData,
            ),
            clientDataJson: base64Encode(assertion.response.clientDataJSON),
            signature: base64Encode(assertion.response.signature),
            userHandle: base64Encode(
              assertion.response.userHandle || new ArrayBuffer(0),
            ),
            type: "public-key",
          },
        };
        request.onCredential(assertionCredential);
      })
      .catch((error: Error) => {
        request.onAssertionError(error);
      });
  },

  startAttestation(request: WebauthnAttestationRequest): void {
    const options = request.enroll.options;
    const user = options.user;
    const opts: PublicKeyCredentialCreationOptions = {
      rp: options.rp,
      user: Object.assign({}, user, { id: base64Decode(user.id) }),
      challenge: base64Decode(options.challenge),
      pubKeyCredParams: options.pubKeyCredParams.map((p) => ({
        type: p.type as PublicKeyCredentialType,
        alg: p.alg,
      })),
      attestation: options.attestation as AttestationConveyancePreference,
      timeout: options.timeout,
      excludeCredentials: options.excludeCredentials.map((c) => ({
        type: c.type as PublicKeyCredentialType,
        id: base64Decode(c.id),
        transports: c.transports as AuthenticatorTransport[],
      })),
      authenticatorSelection: options.authenticatorSelection
        ? {
            authenticatorAttachment: options.authenticatorSelection
              .authenticatorAttachment as AuthenticatorAttachment,
            requireResidentKey:
              options.authenticatorSelection.requireResidentKey,
            residentKey: options.authenticatorSelection
              .residentKey as ResidentKeyRequirement,
            userVerification: options.authenticatorSelection
              .userVerification as UserVerificationRequirement,
          }
        : undefined,
    };
    const navigatorObj = window.navigator;
    if (!navigatorObj || !navigatorObj.credentials) {
      request.onNotSupported();
      return;
    }
    navigatorObj.credentials
      .create({ publicKey: opts })
      .then((credential: Credential | null) => {
        const attestation = credential as PublicKeyCredential & {
          response: AuthenticatorAttestationResponse & {
            getTransports(): AuthenticatorTransport[];
          };
        };
        const attestationResponse: ApiAuthenticatorAttestationResponse = {
          id: base64Encode(attestation.rawId),
          attestationObject: base64Encode(
            attestation.response.attestationObject,
          ),
          clientDataJson: base64Encode(attestation.response.clientDataJSON),
          transports: attestation.response.getTransports(),
        };
        request.onCredential(attestationResponse);
      })
      .catch((error: Error) => {
        request.onError(error);
      });
  },
};
