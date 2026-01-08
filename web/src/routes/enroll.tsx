import { useState } from "react";
import { useApiService } from "../api/api_client";
import {
  ApiAuthenticatorAttestationResponse,
  ApiCredential,
  ApiEnrollStartError,
} from "../api/api_types";
import { useWebauthnService } from "../lib/webauthn-hook";
import {
  IconCheck,
  IconExclamationCircle,
  IconLoader2,
  IconDeviceDesktop,
} from "@tabler/icons-react";
import { Link } from "react-router";

type StartState = {
  state: "start";
};
type EnrollLoading = {
  state: "loading";
};
type EnrollRequestAttestation = {
  state: "request-attestation";
};

type EnrollError = {
  state: "error";
  startError?: ApiEnrollStartError;
};

type EnrollName = {
  state: "name";
  token: string;
  attestationResponse: ApiAuthenticatorAttestationResponse;
  credential: ApiCredential;
};

type EnrollDone = {
  state: "done";
};

type State =
  | StartState
  | EnrollLoading
  | EnrollRequestAttestation
  | EnrollName
  | EnrollDone
  | EnrollError;

function assertUnreachable(value: never): never {
  throw new Error(`Didn't expect to get here: ${value}`);
}

export default function Enroll() {
  const api = useApiService();
  const webauthn = useWebauthnService();
  const [state, setState] = useState<State>({ state: "start" });
  const [name, setName] = useState<string>("Unnamed passkey");

  const startEnrollment = () => {
    setState({ state: "loading" });
    api
      .StartEnroll()
      .then((res) => {
        if (res.error) {
          setState({ state: "error", startError: res.error });
        } else if (res.enrollRequest) {
          setState({ state: "request-attestation" });
          const token = res.enrollRequest.token;
          webauthn.startAttestation({
            enroll: res.enrollRequest,
            onCredential: function (
              attestationResponse: ApiAuthenticatorAttestationResponse,
            ): void {
              console.log("Attestation credential: ", attestationResponse);
              api
                .FinishEnroll({
                  token,
                  attestationResponse,
                })
                .then((res) => {
                  if (res.credential) {
                    setName(res.credential.name);
                    setState({
                      state: "name",
                      token,
                      attestationResponse,
                      credential: res.credential,
                    });
                  } else {
                    setState({ state: "error" });
                  }
                })
                .catch(() => {
                  setState({ state: "error" });
                });
            },
            onError: function (): void {
              console.log("Attestation error occurred");
              setState({ state: "error" });
            },
            onNotSupported: function (): void {
              console.log("Attestation not supported");
              setState({ state: "error" });
            },
          });
        }
      })
      .catch(() => {
        setState({ state: "error" }); // Catch network or other unexpected errors
      });
  };

  const updateCredentialName = () => {
    if (state.state === "name") {
      api
        .UpdateCredential(state.credential.id, { name })
        .then(() => setState({ state: "done" }))
        .catch(() => setState({ state: "error" })); // Catch network or other unexpected errors
    }
  };

  const renderContent = (() => {
    switch (state.state) {
      case "start":
        return (
          <div className="space-y-6">
            <p className="text-slate-600">
              Enroll a new passkey to securely access your account. This will
              allow you to sign in without a password.
            </p>
            <button
              type="button"
              onClick={startEnrollment}
              className="flex justify-center w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-sm bg-emerald-600 hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
            >
              Start Enrollment
            </button>
          </div>
        );
      case "loading":
        return (
          <div className="flex flex-col items-center justify-center space-y-4 py-8">
            <IconLoader2 className="animate-spin text-emerald-500" size={48} />
            <p className="text-lg text-slate-700">Loading...</p>
          </div>
        );
      case "request-attestation":
        return (
          <div className="flex flex-col items-center justify-center space-y-4 py-8 text-center">
            <p className="text-lg text-slate-700">
              Please touch your security key or use your device's biometric
              sensor to enroll your passkey.
            </p>
            <IconDeviceDesktop className="text-slate-500" size={64} />{" "}
            {/* Placeholder icon */}
          </div>
        );
      case "error":
        return (
          <div className="flex flex-col items-center justify-center space-y-4 py-8 text-center">
            <IconExclamationCircle className="text-red-500" size={48} />
            <p className="text-lg text-red-700">
              An error occurred during enrollment. Please try again.
            </p>
            <button
              type="button"
              onClick={() => setState({ state: "start" })}
              className="px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-sm bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
            >
              Try Again
            </button>
          </div>
        );
      case "name":
        return (
          <div className="space-y-6">
            <div>
              <label
                htmlFor="passkey-name"
                className="block text-sm font-medium text-slate-700"
              >
                Passkey Nickname
              </label>
              <div className="mt-1">
                <input
                  id="passkey-name"
                  type="text"
                  className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-sm appearance-none focus:outline-none focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  autoFocus={true}
                  required={true}
                  placeholder="e.g., My Laptop, YubiKey"
                />
              </div>
              <p className="mt-2 text-sm text-slate-500">
                Your passkey has been successfully saved! You can now give it a
                name to help you identify it later, or skip this step.
              </p>
            </div>
            <div className="flex space-x-4">
              <button
                type="submit"
                onClick={updateCredentialName}
                className="flex-1 px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-sm bg-emerald-600 hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
              >
                Finish
              </button>
            </div>
          </div>
        );
      case "done":
        return (
          <div className="flex flex-col items-center justify-center space-y-4 py-8 text-center">
            <IconCheck className="text-emerald-500" size={48} />
            <p className="text-lg text-slate-700">
              Passkey enrolled successfully!
            </p>
            <Link
              to="/"
              className="px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-sm bg-emerald-600 hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
            >
              Go to Dashboard
            </Link>
          </div>
        );
    }
    return assertUnreachable(state);
  })();

  return (
    <section className="bg-gray-50 min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="w-full max-w-md bg-white rounded-lg shadow-lg md:mt-0 xl:p-0">
        <div className="p-6 space-y-6 sm:p-8">
          <h1 className="text-2xl font-bold leading-tight tracking-tight text-slate-800 md:text-3xl text-center">
            Enroll New Passkey
          </h1>
          {renderContent}
        </div>
      </div>
    </section>
  );
}
