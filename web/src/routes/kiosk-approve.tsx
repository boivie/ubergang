import {
  IconAlertTriangle,
  IconCheck,
  IconClock,
  IconDeviceDesktop,
  IconDeviceMobile,
  IconExclamationCircle,
  IconGlobe,
  IconLoader2,
  IconWorld,
} from "@tabler/icons-react";
import { useState } from "react";
import { useLoaderData } from "react-router";
import { ApiService, NotSignedInError, useApiService } from "../api/api_client";
import {
  ApiAssertionCredential,
  ApiQueryKioskSigninResponse,
} from "../api/api_types";
import { useWebauthnService } from "../lib/webauthn-hook";

type LoaderData =
  | { type: "found"; res: ApiQueryKioskSigninResponse; code: string }
  | { type: "not-signed-in"; code: string }
  | { type: "failed" };

export async function KioskApproveLoader(
  api: ApiService,
  code: string,
): Promise<LoaderData> {
  try {
    const res = await api.QueryKioskSignin({ code });
    return { type: "found", res, code };
  } catch (e) {
    if (e instanceof NotSignedInError) {
      return { type: "not-signed-in", code };
    }
    console.log("Failed to look up kiosk sign-in", e);
    return { type: "failed" };
  }
}

const DURATIONS = [
  { label: "1 day", seconds: 24 * 60 * 60 },
  { label: "30 days", seconds: 30 * 24 * 60 * 60 },
  { label: "Until revoked", seconds: 0 },
];
const DEFAULT_DURATION_SECONDS = 30 * 24 * 60 * 60;

type State =
  | { type: "review" }
  | { type: "authenticating" }
  | { type: "success" }
  | { type: "error"; message: string };

function Page({ children }: { children: React.ReactNode }) {
  return (
    <section className="bg-gray-50 min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="w-full max-w-md bg-white rounded-lg shadow-lg md:mt-0 xl:p-0">
        <div className="p-6 space-y-6 sm:p-8">
          <div className="text-center">
            <h1 className="text-2xl font-bold leading-tight tracking-tight text-slate-800 md:text-3xl">
              Sign In a Device
            </h1>
          </div>
          {children}
        </div>
      </div>
    </section>
  );
}

function ErrorMessage({
  message,
  action,
}: {
  message: string;
  action?: { label: string; href: string };
}) {
  return (
    <div className="space-y-6">
      <div className="flex flex-col items-center space-y-4 py-4">
        <IconExclamationCircle className="text-red-500" size={48} />
        <p className="text-lg text-red-700 text-center">{message}</p>
      </div>
      {action && (
        <a
          href={action.href}
          className="block w-full px-4 py-2 text-sm font-medium text-white text-center border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
        >
          {action.label}
        </a>
      )}
    </div>
  );
}

function Detail({
  icon,
  label,
  children,
}: {
  icon: React.ReactNode;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-start space-x-3">
      <div className="text-slate-500 mt-0.5 shrink-0">{icon}</div>
      <div className="min-w-0">
        <dt className="text-xs font-medium text-slate-600 uppercase tracking-wide">
          {label}
        </dt>
        <dd className="text-sm text-slate-800 break-all">{children}</dd>
      </div>
    </div>
  );
}

// Where a signed-in user reviews and approves a kiosk's sign-in request.
export default function KioskApprove() {
  const data = useLoaderData() as LoaderData;

  if (data.type === "not-signed-in") {
    return (
      <Page>
        <ErrorMessage
          message="You need to be signed in on this device to approve another device."
          action={{
            label: "Sign in",
            href: "/signin?rd=" + encodeURIComponent("/kiosk/" + data.code),
          }}
        />
      </Page>
    );
  }
  if (data.type === "failed") {
    return (
      <Page>
        <ErrorMessage
          message="Something went wrong. Please try again."
          action={{ label: "Try again", href: "/kiosk/" }}
        />
      </Page>
    );
  }

  const error = data.res.error;
  if (error) {
    let message = "Something went wrong. Please try again.";
    let action = { label: "Enter another code", href: "/kiosk/" };
    if (error.invalidCode) {
      message =
        "This code is invalid or has expired. Check the code shown on the device.";
    } else if (error.notAllowed) {
      message =
        "You don't have access to the site that this device wants to sign in to.";
    } else if (error.noCredentials) {
      message = "You need a passkey to approve sign-ins.";
      action = { label: "Create a passkey", href: "/enroll/" };
    } else if (error.tooManyAttempts) {
      message =
        "Too many invalid codes have been entered. Wait a few minutes and try again.";
    }
    return (
      <Page>
        <ErrorMessage message={message} action={action} />
      </Page>
    );
  }

  return <Approve res={data.res} />;
}

function Approve({ res }: { res: ApiQueryKioskSigninResponse }) {
  const api = useApiService();
  const webauthn = useWebauthnService();
  const [state, setState] = useState<State>({ type: "review" });
  const [name, setName] = useState("");
  const [durationSeconds, setDurationSeconds] = useState(
    DEFAULT_DURATION_SECONDS,
  );

  const approve = () => {
    setState({ type: "authenticating" });
    webauthn.startAssertion({
      request: res.assertionRequest!,
      onCredential: async (credential: ApiAssertionCredential) => {
        try {
          const response = await api.ApproveKioskSignin({
            token: res.token!,
            credential,
            name,
            durationSeconds,
          });
          if (!response.error) {
            setState({ type: "success" });
          } else if (response.error.expired) {
            setState({
              type: "error",
              message:
                "The sign-in request expired. The device is showing a new code.",
            });
          } else {
            setState({
              type: "error",
              message: "Failed to approve the sign-in. Please try again.",
            });
          }
        } catch (_error) {
          setState({
            type: "error",
            message: "Failed to approve the sign-in. Please try again.",
          });
        }
      },
      onAssertionError: (error: Error) => {
        console.error("WebAuthn assertion error:", error);
        setState({
          type: "error",
          message: "Authentication was cancelled or failed. Please try again.",
        });
      },
      onNotSupported: () => {
        setState({
          type: "error",
          message: "Passkeys aren't supported on this device or browser.",
        });
      },
    });
  };

  switch (state.type) {
    case "authenticating":
      return (
        <Page>
          <div className="flex flex-col items-center justify-center space-y-4 py-8">
            <IconLoader2 className="animate-spin text-emerald-500" size={48} />
            <p className="text-lg text-slate-700">Authenticating...</p>
            <p className="text-sm text-slate-500 text-center">
              Use your passkey to approve the sign-in.
            </p>
          </div>
        </Page>
      );
    case "success":
      return (
        <Page>
          <div className="flex flex-col items-center justify-center space-y-4 py-8">
            <IconCheck className="text-emerald-500" size={48} />
            <p className="text-lg text-emerald-700 font-medium text-center">
              The device is now signed in to {res.host}.
            </p>
            <p className="text-sm text-slate-600 text-center">
              You can revoke its access from your sessions on the dashboard.
            </p>
          </div>
        </Page>
      );
    case "error":
      // The approval token can only be used once, so start over.
      return (
        <Page>
          <ErrorMessage
            message={state.message}
            action={{ label: "Try again", href: window.location.pathname }}
          />
        </Page>
      );
  }

  return (
    <Page>
      <div className="space-y-6">
        <p className="text-slate-600 text-center">
          A device wants to sign in to <b className="break-all">{res.host}</b>{" "}
          as you. It won't be able to access anything else.
        </p>

        <dl className="bg-slate-50 rounded-lg p-4 space-y-3">
          <Detail icon={<IconWorld size={16} />} label="Site">
            {res.host}
          </Detail>
          <Detail icon={<IconGlobe size={16} />} label="Device IP address">
            <span className="font-mono">{res.requestorIp}</span>
          </Detail>
          <Detail icon={<IconDeviceMobile size={16} />} label="Device">
            {res.requestorUserAgent}
          </Detail>
          {res.requestedAt && (
            <Detail icon={<IconClock size={16} />} label="Requested">
              {new Date(res.requestedAt).toLocaleString()}
            </Detail>
          )}
        </dl>

        {res.sameNetwork ? (
          <div className="bg-emerald-50 border border-emerald-200 rounded-lg p-4">
            <p className="text-sm text-emerald-800">
              The device seems to be on the same network as you.
            </p>
          </div>
        ) : (
          <div
            className="bg-amber-50 border border-amber-200 rounded-lg p-4 flex space-x-3"
            role="alert"
          >
            <IconAlertTriangle
              className="text-amber-600 shrink-0 mt-0.5"
              size={20}
            />
            <p className="text-sm text-amber-800">
              <strong>The device is on a different network than you</strong> (
              <span className="font-mono">{res.approverIp}</span>). This can
              happen if one of you uses mobile data or IPv6. Only approve if you
              can see the device and it's showing this code. If someone sent you
              the code, don't approve it.
            </p>
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label
              htmlFor="kiosk-name"
              className="block text-sm font-medium text-slate-700"
            >
              Name (optional)
            </label>
            <input
              id="kiosk-name"
              type="text"
              maxLength={100}
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Kitchen tablet"
              className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm shadow-xs focus:border-emerald-500 focus:outline-hidden focus:ring-1 focus:ring-emerald-500"
            />
          </div>
          <fieldset>
            <legend className="block text-sm font-medium text-slate-700">
              Stay signed in for
            </legend>
            <div className="mt-2 grid grid-cols-3 gap-2">
              {DURATIONS.map((d) => (
                <label
                  key={d.seconds}
                  className={
                    "cursor-pointer rounded-md border px-3 py-2 text-center text-sm " +
                    (durationSeconds === d.seconds
                      ? "border-emerald-500 bg-emerald-50 text-emerald-800 font-medium"
                      : "border-slate-300 text-slate-700 hover:bg-slate-50")
                  }
                >
                  <input
                    type="radio"
                    name="duration"
                    className="sr-only"
                    checked={durationSeconds === d.seconds}
                    onChange={() => setDurationSeconds(d.seconds)}
                  />
                  {d.label}
                </label>
              ))}
            </div>
          </fieldset>
        </div>

        <button
          type="button"
          onClick={approve}
          className="w-full px-4 py-3 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
        >
          <div className="flex items-center justify-center space-x-2">
            <IconDeviceDesktop size={18} />
            <span>Approve with Passkey</span>
          </div>
        </button>
      </div>
    </Page>
  );
}
