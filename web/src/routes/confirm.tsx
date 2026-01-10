import {
  IconDeviceMobile,
  IconExclamationCircle,
  IconLoader2,
} from "@tabler/icons-react";
import { useState } from "react";
import { Form } from "react-router";
import { useApiService } from "../api/api_client";
import OTPInput from "../components/otp_input";

type State =
  | { type: "input" }
  | { type: "loading" }
  | { type: "error"; message: string };

export default function Confirm() {
  const api = useApiService();
  const [pin, setPin] = useState("");
  const [state, setState] = useState<State>({ type: "input" });

  const handleSubmitPin = async (pin: string) => {
    if (pin.length !== 6) {
      setState({
        type: "error",
        message: "Please enter a complete 6-digit PIN.",
      });
      return;
    }

    setState({ type: "loading" });

    try {
      const res = await api.QuerySigninPin({ pin });

      if (res.error) {
        setState({
          type: "error",
          message: res.error.invalidPin
            ? "Invalid PIN. Please check the PIN and try again."
            : "Unable to verify PIN. Please try again.",
        });
      } else {
        window.location.href = "/confirm/" + pin;
      }
    } catch (error) {
      setState({
        type: "error",
        message: "Network error. Please check your connection and try again.",
      });
    }
  };

  const renderContent = () => {
    switch (state.type) {
      case "input":
        return (
          <div className="space-y-6">
            <div className="text-center">
              <IconDeviceMobile
                className="mx-auto text-emerald-500 mb-4"
                size={48}
              />
              <p className="text-slate-600">
                Enter the 6-digit PIN from your sign-in request to continue.
              </p>
            </div>
            <Form className="space-y-6" method="POST">
              <div>
                <label htmlFor="pin" className="sr-only">
                  Confirmation PIN
                </label>
                <OTPInput
                  numInputs={6}
                  onChange={(otp) => setPin(otp)}
                  value={pin}
                  shouldAutoFocus
                />
                <p className="mt-3 text-sm text-slate-500 text-center">
                  Check your other device or the sign-in page for the PIN
                </p>
              </div>
              <button
                type="submit"
                onClick={(e) => {
                  e.preventDefault();
                  handleSubmitPin(pin);
                }}
                disabled={pin.length !== 6}
                className="w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Confirm Sign In
              </button>
            </Form>
          </div>
        );

      case "loading":
        return (
          <div className="flex flex-col items-center justify-center space-y-4 py-8">
            <IconLoader2 className="animate-spin text-emerald-500" size={48} />
            <p className="text-lg text-slate-700">Verifying PIN...</p>
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
            <button
              type="button"
              onClick={() => {
                setState({ type: "input" });
                setPin("");
              }}
              className="w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-red-600 hover:bg-red-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
            >
              Try Again
            </button>
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
            <p className="mt-2 text-sm text-slate-600">Secure Proxy</p>
          </div>
          {renderContent()}
        </div>
      </div>
    </section>
  );
}
