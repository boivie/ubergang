import { IconDeviceTv, IconLoader2 } from "@tabler/icons-react";
import { useEffect, useState } from "react";
import { useApiService } from "../api/api_client";
import { ApiRequestKioskSigninResponse } from "../api/api_types";

type PendingRequest = NonNullable<ApiRequestKioskSigninResponse["success"]>;

type State =
  | { type: "loading" }
  // The backend doesn't allow kiosk sign-in.
  | { type: "unavailable" }
  // The request couldn't be made, and will be retried.
  | { type: "retrying" }
  | { type: "pending"; request: PendingRequest };

const POLL_INTERVAL_MS = 2000;
const RETRY_INTERVAL_MS = 30000;

// Formats a code like "123 456", to make it easier to read from a distance.
function formatCode(code: string): string {
  return code.replace(/^(\d{3})(\d{3})$/, "$1 $2");
}

function hostOf(url: string): string {
  try {
    return new URL(url).hostname;
  } catch {
    return url;
  }
}

// Lets a device that can't sign in by itself, such as a wall-mounted tablet,
// be signed in to the backend at `rd` from another device. It shows a code
// that a signed-in user approves, and then redirects to `rd`.
//
// This needs no input on the device: new codes are requested when they
// expire, and errors are retried.
export function KioskSignin({ rd }: { rd: string }) {
  const api = useApiService();
  const [state, setState] = useState<State>({ type: "loading" });

  useEffect(() => {
    let cancelled = false;
    let timer: number | undefined;

    const schedule = (fn: () => void, delayMs: number) => {
      if (!cancelled) {
        timer = window.setTimeout(fn, delayMs);
      }
    };

    const request = async () => {
      try {
        const res = await api.RequestKioskSignin({ redirect: rd });
        if (cancelled) return;
        if (res.success) {
          const id = res.success.id;
          setState({ type: "pending", request: res.success });
          schedule(() => poll(id), POLL_INTERVAL_MS);
        } else if (res.error?.notAllowed) {
          setState({ type: "unavailable" });
        } else {
          setState({ type: "retrying" });
          schedule(request, RETRY_INTERVAL_MS);
        }
      } catch (e) {
        console.log("Failed to request kiosk sign-in", e);
        if (cancelled) return;
        setState({ type: "retrying" });
        schedule(request, RETRY_INTERVAL_MS);
      }
    };

    const poll = async (id: string) => {
      try {
        const res = await api.PollKioskSignin({ id });
        if (cancelled) return;
        if (res.redirect) {
          window.location.replace(res.redirect);
        } else if (res.expired) {
          request();
        } else {
          schedule(() => poll(id), POLL_INTERVAL_MS);
        }
      } catch (e) {
        // The server may be restarting. Once it's back, the request will have
        // expired, and a new one is made.
        console.log("Failed to poll kiosk sign-in", e);
        schedule(() => poll(id), RETRY_INTERVAL_MS);
      }
    };

    request();
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [api, rd]);

  if (state.type === "unavailable") {
    return null;
  }

  return (
    <div className="border-t border-slate-200 pt-6 space-y-4">
      <div className="flex items-center justify-center space-x-2">
        <IconDeviceTv className="text-slate-600" size={20} />
        <h2 className="text-lg font-medium text-slate-800">
          Or sign in from another device
        </h2>
      </div>
      {state.type === "pending" ? (
        <>
          <p className="text-sm text-slate-600 text-center">
            On a device where you're signed in, scan the QR code, or go to{" "}
            <b className="break-all">{state.request.confirmUrl}</b> and enter
            the code.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-6">
            <div className="bg-white p-2 rounded-lg shadow-xs border shrink-0">
              <img
                src={state.request.qrCodeUrl}
                alt="QR code for approving this sign-in"
                className="w-40 h-40"
              />
            </div>
            <div
              id="kiosk-code"
              aria-label="Sign-in code"
              className="font-mono font-bold text-4xl tracking-widest text-slate-800 whitespace-nowrap"
            >
              {formatCode(state.request.code)}
            </div>
          </div>
          <div className="flex items-center justify-center space-x-2 text-sm text-slate-500">
            <IconLoader2 className="animate-spin text-emerald-500" size={16} />
            <span>Waiting for approval…</span>
          </div>
          <p className="text-xs text-slate-500 text-center">
            This device will only be signed in to <b>{hostOf(rd)}</b>.
          </p>
        </>
      ) : (
        <div className="flex items-center justify-center space-x-2 text-sm text-slate-500 py-4">
          <IconLoader2 className="animate-spin text-emerald-500" size={16} />
          <span>
            {state.type === "loading"
              ? "Getting a sign-in code…"
              : "Couldn't get a sign-in code. Retrying soon…"}
          </span>
        </div>
      )}
    </div>
  );
}
