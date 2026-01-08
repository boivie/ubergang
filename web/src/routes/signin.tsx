import { useState } from "react";
import { useApiService } from "../api/api_client";
import {
  ApiAssertionCredential,
  ApiSignInEmailError,
  ApiSignInEmailSuccess,
  ApiSignInWebauthResponse,
  ApiSigninEmailResponse,
} from "../api/api_types";
import { EmailForm } from "../components/email_form";
import { useWebauthnService } from "../lib/webauthn-hook";
import {
  IconLoader2,
  IconExclamationCircle,
  IconDeviceDesktop,
} from "@tabler/icons-react";

type SignInEmail = {
  state: "email";
};

type SignInLoading = {
  state: "loading";
};

type RequestAssertionState = {
  state: "request-assertion";
  info: ApiSignInEmailSuccess;
  email: string;
};

type SigninEmailError = {
  state: "email-error";
  error: ApiSignInEmailError;
};

type RequestSigninError = {
  state: "request-signin-error";
};

type State =
  | SignInEmail
  | SignInLoading
  | RequestAssertionState
  | SigninEmailError
  | RequestSigninError;

function handleSignedIn(res: ApiSignInWebauthResponse) {
  if (!res.success) {
    alert("Failed to log in");
    return;
  }
  if (res.success.cookie) {
    document.cookie = res.success.cookie;
  }
  if (res.success.redirect) {
    window.location.replace(res.success.redirect);
  } else {
    window.location.replace("/");
  }
}

function assertUnreachable(value: never): never {
  throw new Error(`Didn't expect to get here: ${value}`);
}

export default function SignIn() {
  const api = useApiService();
  const webauthn = useWebauthnService();

  const [state, setState] = useState<State>({ state: "email" });

  const startSigninPinFlow = (email: string) => {
    api
      .RequestSigninPin({
        email,
        userAgent: window.navigator.userAgent,
      })
      .then((res) => {
        if (res.error) {
          console.log("Failed to request signin pin", res.error);
          setState({ state: "request-signin-error" });
        } else {
          window.location.href = "/signin/" + res.id;
        }
      });
  };

  const handleConditionalWebauthn = async () => {
    if (
      !PublicKeyCredential.isConditionalMediationAvailable ||
      !(await PublicKeyCredential.isConditionalMediationAvailable())
    ) {
      return;
    }

    const res = await api.GetPasswordlessToken();
    webauthn.startConditionalAssertion({
      request: res.assertionRequest!,
      onCredential: (credential) => {
        handleSigninCredential(res.token!, credential);
      },
      onAssertionError: (error: Error) => {
        console.log("Failed conditional assertion", error);
      },
      onNotSupported: () => {},
    });
  };

  const handleSubmitEmail = (email: string) => {
    setState({ state: "loading" });
    api
      .SignInEmail(email)
      .then((res: ApiSigninEmailResponse) => {
        if (res.success) {
          const success = res.success;
          setState({ state: "request-assertion", info: res.success, email });
          webauthn.startAssertion({
            request: success.assertionRequest,
            onCredential: (credential) =>
              handleSigninCredential(success.token, credential),
            onAssertionError: () => startSigninPinFlow(email),
            onNotSupported: () => startSigninPinFlow(email),
          });
        } else if (res.error) {
          setState({ state: "email-error", error: res.error });
        }
      })
      .catch((e) => {
        console.log("Failed to sign in", e);
        setState({ state: "request-signin-error" });
      });
  };

  const handleSigninCredential = async (
    token: string,
    credential: ApiAssertionCredential,
  ) => {
    const signinResult = await api.SignInWebauthn({
      token,
      credential,
      redirect: "/",
    });
    handleSignedIn(signinResult);
  };

  const renderErrorMessage = (error?: ApiSignInEmailError) => {
    if (!error) return "An error occurred while signing in. Please try again.";

    if (error.internal_error) {
      return "An internal error occurred while signing in. Please try again.";
    } else if (error.wrong_email) {
      return "The email address you entered is invalid. Please check your email and try again.";
    } else if (error.no_credentials) {
      return "The account hasn't been properly set up yet";
    }
    return "An error occurred while signing in. Please try again.";
  };

  const component = (() => {
    switch (state.state) {
      case "email":
        return (
          <div className="space-y-6">
            <p className="text-slate-600 text-center">
              Enter your email address to sign in with your passkey or receive a
              secure sign-in link.
            </p>
            <EmailForm
              onSubmit={handleSubmitEmail}
              onConditionalWebauthn={handleConditionalWebauthn}
            />
          </div>
        );
      case "loading":
        return (
          <div className="flex flex-col items-center justify-center space-y-4 py-8">
            <IconLoader2 className="animate-spin text-emerald-500" size={48} />
            <p className="text-lg text-slate-700">Signing you in...</p>
          </div>
        );
      case "request-assertion":
        return (
          <div className="flex flex-col items-center justify-center space-y-4 py-8 text-center">
            <IconDeviceDesktop className="text-emerald-500" size={64} />
            <p className="text-lg text-slate-700 font-medium">
              Use your passkey to sign in
            </p>
            <p className="text-sm text-slate-500 max-w-xs">
              Touch your security key or use your device's biometric sensor.
            </p>

            <div className="pt-4 w-full max-w-xs">
              <button
                onClick={() => startSigninPinFlow(state.email)}
                className="w-full px-4 py-2 text-sm font-medium text-emerald-700 bg-emerald-100 border border-transparent rounded-md hover:bg-emerald-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
              >
                Sign in with another device
              </button>
            </div>
          </div>
        );
      case "email-error":
        return (
          <div className="space-y-6">
            <div className="flex flex-col items-center space-y-4 py-4">
              <IconExclamationCircle className="text-red-500" size={48} />
              <p className="text-lg text-red-700 text-center">
                {renderErrorMessage(state.error)}
              </p>
            </div>
            <button
              type="button"
              onClick={() => setState({ state: "email" })}
              className="w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-sm bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
            >
              Try Again
            </button>
          </div>
        );
      case "request-signin-error":
        return (
          <div className="space-y-6">
            <div className="flex flex-col items-center space-y-4 py-4">
              <IconExclamationCircle className="text-red-500" size={48} />
              <p className="text-lg text-red-700 text-center">
                Unable to process your sign-in request. Please check your
                connection and try again.
              </p>
            </div>
            <button
              type="button"
              onClick={() => setState({ state: "email" })}
              className="w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-sm bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
            >
              Try Again
            </button>
          </div>
        );
    }
    return assertUnreachable(state);
  })();

  return (
    <section className="bg-gray-50 min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="w-full max-w-md bg-white rounded-lg shadow-lg md:mt-0 xl:p-0">
        <div className="p-6 space-y-6 sm:p-8">
          <div className="text-center">
            <h1 className="text-2xl font-bold leading-tight tracking-tight text-slate-800 md:text-3xl">
              Sign In
            </h1>
            <p className="mt-2 text-sm text-slate-600">Secure Proxy</p>
          </div>
          {component}
        </div>
      </div>
    </section>
  );
}
