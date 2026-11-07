import { ApiSession } from "../api/api_types";
import { sessionLimits } from "../lib/session_utils";

export function KioskBadge({ session }: { session: ApiSession }) {
  if (!session.restrictedHost) {
    return null;
  }
  return (
    <span className="ml-2 text-xs text-white bg-amber-500 rounded-full px-2 py-1">
      Kiosk
    </span>
  );
}

export function SessionLimits({ session }: { session: ApiSession }) {
  const limits = sessionLimits(session);
  if (!limits) {
    return null;
  }
  return <p className="w-full truncate text-sm text-slate-500">{limits}</p>;
}
