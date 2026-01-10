import {
  IconDeviceMobile,
  IconExclamationCircle,
  IconLoader2,
  IconQrcode,
} from "@tabler/icons-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { redirect, useLoaderData, useParams } from "react-router";
import { ApiService, useApiService } from "../api/api_client";
import { ApiPollSigninPinResponse } from "../api/api_types";

export async function SigninTokenLoader(api: ApiService, token: string) {
  const res = await api.PollLoginRequest({ id: token, redirect: "/" });
  if (res.success) {
    if (res.success.cookie) {
      document.cookie = res.success.cookie;
    }
    return redirect(res.success.redirect);
  }
  return res;
}

export default function SigninToken() {
  const api = useApiService();
  const initialRes = useLoaderData() as ApiPollSigninPinResponse;
  const { token } = useParams();
  const [res, setRes] = useState<ApiPollSigninPinResponse>(initialRes);
  const [isPollingTimeout, setIsPollingTimeout] = useState<boolean>(false);
  const intervalRef = useRef<number | null>(null);
  const lastSuccessfulPollRef = useRef<number>(0);

  useEffect(() => {
    lastSuccessfulPollRef.current = Date.now();
  }, []);

  const pollRequest = useCallback(async () => {
    try {
      const response = await api.PollLoginRequest({
        id: token!,
        redirect: "/",
      });

      lastSuccessfulPollRef.current = Date.now();
      setRes(response);

      if (response.error) {
        clearInterval(intervalRef.current!);
        intervalRef.current = null;
      } else if (response.success) {
        clearInterval(intervalRef.current!);
        intervalRef.current = null;
        if (response.success.cookie) {
          document.cookie = response.success.cookie;
        }
        if (response.success.redirect) {
          window.location.replace(response.success.redirect);
        } else {
          window.location.replace("/");
        }
      }
    } catch (error) {
      console.error("Polling error:", error);
    }
  }, [api, token]);

  const startPolling = useCallback(() => {
    if (intervalRef.current) return;

    lastSuccessfulPollRef.current = Date.now();

    intervalRef.current = setInterval(() => {
      const timeSinceLastSuccess = Date.now() - lastSuccessfulPollRef.current;
      if (timeSinceLastSuccess > 15000) {
        setIsPollingTimeout(true);
        clearInterval(intervalRef.current!);
        intervalRef.current = null;
      } else {
        pollRequest();
      }
    }, 1000);
  }, [pollRequest]);

  const stopPolling = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (res.pending && !res.error && !res.success && !intervalRef.current) {
      startPolling();
    }

    return stopPolling;
  }, [res.pending, res.error, res.success, startPolling, stopPolling]);

  function retry() {
    setIsPollingTimeout(false);
    startPolling();
  }

  let component = (
    <div className="flex flex-col items-center space-y-4 py-8">
      <IconExclamationCircle className="text-red-500" size={48} />
      <p className="text-lg text-red-700 text-center">
        An unexpected error occurred. Please try signing in again.
      </p>
    </div>
  );

  if (isPollingTimeout) {
    component = (
      <div className="space-y-6">
        <div className="flex flex-col items-center space-y-4 py-4">
          <IconExclamationCircle className="text-orange-500" size={48} />
          <div className="text-center">
            <p className="text-lg text-orange-700 font-medium">
              Connection timeout
            </p>
            <p className="text-sm text-slate-500 mt-2">
              Unable to connect to the server. Please check your connection and
              try again.
            </p>
          </div>
        </div>
        <button
          onClick={retry}
          className="block w-full px-4 py-2 text-sm font-medium text-white text-center border border-transparent rounded-md shadow-xs bg-orange-600 hover:bg-orange-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-orange-500"
        >
          Retry
        </button>
      </div>
    );
  } else if (res.error) {
    let errorTitle = "An error occurred";
    let errorMessage = "Please try signing in again.";

    if (res.error.expired) {
      errorTitle = "Sign-in link expired";
      errorMessage =
        "This sign-in link is no longer valid. Please request a new one.";
    } else if (res.error.invalidToken) {
      errorTitle = "Invalid token";
      errorMessage = "The sign-in token is invalid or has been tampered with.";
    } else if (res.error.internalError) {
      errorTitle = "Server error";
      errorMessage = "An internal server error occurred. Please try again.";
    }

    component = (
      <div className="space-y-6">
        <div className="flex flex-col items-center space-y-4 py-4">
          <IconExclamationCircle className="text-red-500" size={48} />
          <div className="text-center">
            <p className="text-lg text-red-700 font-medium">{errorTitle}</p>
            <p className="text-sm text-slate-500 mt-2">{errorMessage}</p>
          </div>
        </div>
        <a
          href="/signin"
          className="block w-full px-4 py-2 text-sm font-medium text-white text-center border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
        >
          Sign in again
        </a>
      </div>
    );
  } else if (res.pending) {
    const numbers = res.pending.pin.split("");

    component = (
      <div className="space-y-8">
        {/* QR Code Section */}
        <div className="bg-slate-50 rounded-lg p-6 text-center">
          <div className="flex items-center justify-center space-x-2 mb-4">
            <IconQrcode className="text-slate-600" size={20} />
            <h3 className="text-lg font-medium text-slate-800">Scan QR Code</h3>
          </div>
          <p className="text-sm text-slate-600 mb-4">
            Use a signed-in device to scan this QR code:
          </p>
          <div className="flex justify-center mb-4">
            <div className="bg-white p-4 rounded-lg shadow-xs border">
              <img
                src={res.pending.qr_code_url}
                alt="QR Code for sign-in confirmation"
                className="w-48 h-48"
              />
            </div>
          </div>
        </div>

        {/* PIN Section */}
        <div className="bg-slate-50 rounded-lg p-6">
          <div className="flex items-center justify-center space-x-2 mb-4">
            <IconDeviceMobile className="text-slate-600" size={20} />
            <h3 className="text-lg font-medium text-slate-800">Or Enter PIN</h3>
          </div>
          <p className="text-sm text-slate-600 mb-4 text-center">
            On a signed-in device, visit <b>{res.pending.confirm_url}</b> and
            enter this PIN:
          </p>
          <div
            className="flex justify-center space-x-2 mb-4"
            id="pin-container"
          >
            {numbers.map((digit, i) => (
              <div
                key={i}
                className="bg-white w-12 h-12 flex items-center justify-center rounded-lg border-2 border-slate-200 shadow-xs"
              >
                <span className="font-bold text-xl text-slate-800">
                  {digit}
                </span>
              </div>
            ))}
          </div>
        </div>

        <div className="text-center">
          <div className="flex flex-col items-center space-y-4">
            <IconLoader2 className="animate-spin text-emerald-500" size={32} />
            <p className="text-sm text-slate-600">
              Waiting for confirmation...
            </p>
          </div>

          <p className="text-xs text-slate-500">
            This page will automatically update when you confirm the sign-in
            request.
          </p>
        </div>
      </div>
    );
  }

  return (
    <section className="bg-gray-50 min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="w-full max-w-xl bg-white rounded-lg shadow-lg md:mt-0 xl:p-0">
        <div className="p-6 space-y-6 sm:p-8">
          <div className="text-center">
            <h1 className="text-2xl font-bold leading-tight tracking-tight text-slate-800 md:text-3xl">
              Complete Sign In
            </h1>
          </div>
          {component}
        </div>
      </div>
    </section>
  );
}
