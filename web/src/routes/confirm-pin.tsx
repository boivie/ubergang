import {
  IconCheck,
  IconDeviceDesktop,
  IconDeviceMobile,
  IconExclamationCircle,
  IconGlobe,
  IconLoader2,
} from "@tabler/icons-react";
import { useState } from "react";
import { useLoaderData } from "react-router";
import { ApiService, useApiService } from "../api/api_client";
import {
  ApiAssertionCredential,
  ApiQuerySigninPinResponse,
} from "../api/api_types";
import { useWebauthnService } from "../lib/webauthn-hook";

export async function ConfirmPinLoader(api: ApiService, pin: string) {
  return api.QuerySigninPin({
    pin: pin!,
  });
}

type State =
  | { type: "confirm" }
  | { type: "authenticating" }
  | { type: "success" }
  | { type: "error"; message: string };

export default function ConfirmPin() {
  const req = useLoaderData() as ApiQuerySigninPinResponse;
  const webauthn = useWebauthnService();
  const api = useApiService();
  const [state, setState] = useState<State>(
    req.error?.invalidPin
      ? {
          type: "error",
          message: "The PIN is invalid or has expired. Please try again.",
        }
      : { type: "confirm" },
  );

  const startAssertion = async () => {
    setState({ type: "authenticating" });

    try {
      webauthn.startAssertion({
        request: req.assertionRequest!,
        onCredential: async function (
          credential: ApiAssertionCredential,
        ): Promise<void> {
          try {
            const response = await api.ConfirmSigninPin({
              token: req.token!,
              credential: credential,
            });

            if (!response.error) {
              setState({ type: "success" });
            } else {
              setState({
                type: "error",
                message: "Authentication failed. Please try again.",
              });
            }
          } catch (error) {
            setState({
              type: "error",
              message: "Failed to confirm sign-in. Please try again.",
            });
          }
        },
        onAssertionError: function (error: Error): void {
          console.error("WebAuthn assertion error:", error);
          setState({
            type: "error",
            message:
              "Authentication was cancelled or failed. Please try again.",
          });
        },
        onNotSupported: function (): void {
          setState({
            type: "error",
            message: "WebAuthn is not supported on this device or browser.",
          });
        },
      });
    } catch (error) {
      setState({
        type: "error",
        message: "Unable to start authentication. Please try again.",
      });
    }
  };

  const renderContent = () => {
    switch (state.type) {
      case "confirm":
        return (
          <div className="space-y-6">
            <div className="text-center">
              <p className="text-slate-600">
                Confirm this sign-in request using your passkey.
              </p>
            </div>

            {/* Security Information */}
            <div className="bg-slate-50 rounded-lg p-4 space-y-3">
              <h3 className="text-sm font-medium text-slate-800 mb-3">
                Sign-in Request Details
              </h3>

              <div className="flex items-start space-x-3">
                <IconGlobe
                  className="text-slate-500 mt-0.5 flex-shrink-0"
                  size={16}
                />
                <div>
                  <dt className="text-xs font-medium text-slate-600 uppercase tracking-wide">
                    IP Address
                  </dt>
                  <dd className="text-sm text-slate-800 font-mono">
                    {req.requestor_ip}
                  </dd>
                </div>
              </div>

              <div className="flex items-start space-x-3">
                <IconDeviceMobile
                  className="text-slate-500 mt-0.5 flex-shrink-0"
                  size={16}
                />
                <div>
                  <dt className="text-xs font-medium text-slate-600 uppercase tracking-wide">
                    Device
                  </dt>
                  <dd className="text-sm text-slate-800 break-all">
                    {req.requestor_user_agent}
                  </dd>
                </div>
              </div>
            </div>

            {/* Warning */}
            <div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
              <p className="text-sm text-amber-800">
                <strong>Security Check:</strong> Only confirm this request if
                you initiated it from the device and location shown above.
              </p>
            </div>

            <button
              type="button"
              onClick={startAssertion}
              className="w-full px-4 py-3 text-sm font-medium text-white border border-transparent rounded-md shadow-sm bg-emerald-600 hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
            >
              <div className="flex items-center justify-center space-x-2">
                <IconDeviceDesktop size={18} />
                <span>Confirm with Passkey</span>
              </div>
            </button>
          </div>
        );

      case "authenticating":
        return (
          <div className="flex flex-col items-center justify-center space-y-4 py-8">
            <IconLoader2 className="animate-spin text-emerald-500" size={48} />
            <p className="text-lg text-slate-700">Authenticating...</p>
            <p className="text-sm text-slate-500 text-center">
              Please use your passkey, biometric sensor, or security key to
              confirm.
            </p>
          </div>
        );

      case "success":
        return (
          <div className="flex flex-col items-center justify-center space-y-4 py-8">
            <IconCheck className="text-emerald-500" size={48} />
            <p className="text-lg text-emerald-700 font-medium">
              Sign-in confirmed!
            </p>
            <p className="text-sm text-slate-600">
              Please continue on your other device.
            </p>
          </div>
        );

      case "error":
        return (
          <div className="space-y-6">
            <div className="flex flex-col items-center space-y-4 py-4">
              <IconExclamationCircle className="text-red-500" size={48} />
              <p className="text-lg text-red-700 text-center">
                {state.message}
              </p>
            </div>
            <div className="flex space-x-3">
              <button
                type="button"
                onClick={() => {
                  if (req.error?.invalidPin) {
                    window.location.href = "/confirm/";
                  } else {
                    setState({ type: "confirm" });
                  }
                }}
                className="flex-1 px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-sm bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
              >
                {req.error?.invalidPin ? "Retry" : "Try Again"}
              </button>
            </div>
          </div>
        );
    }
  };

  return (
    <section className="bg-gray-50 min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="w-full max-w-md bg-white rounded-lg shadow-lg md:mt-0 xl:p-0">
        <div className="p-6 space-y-6 sm:p-8">
          <div className="text-center">
            <h1 className="text-2xl font-bold leading-tight tracking-tight text-slate-800 md:text-3xl">
              Confirm Sign In
            </h1>
          </div>
          {renderContent()}
        </div>
      </div>
    </section>
  );
}
