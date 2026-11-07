import { ApiSession } from "../api/api_types";

// What to call a session in lists: its name if it was given one when it was
// approved (kiosk sessions), and otherwise its browser and OS.
export function sessionTitle(
  session: ApiSession,
  browser: { name?: string },
  os: { name?: string },
): string {
  return session.name || `${browser.name} on ${os.name}`;
}

// Describes the limits of a session: the only site it's valid on, and when it
// expires. Empty for normal sessions.
export function sessionLimits(session: ApiSession): string {
  const parts = [];
  if (session.restrictedHost) {
    parts.push(`Only ${session.restrictedHost}`);
  }
  if (session.expiresAt) {
    const expires = new Date(session.expiresAt);
    parts.push(
      expires < new Date()
        ? "Expired"
        : `Expires ${expires.toLocaleDateString()}`,
    );
  }
  return parts.join(" · ");
}
