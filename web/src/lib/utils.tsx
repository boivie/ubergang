export function base64Encode(arrayBuffer: ArrayBuffer) {
  const bytes = new Uint8Array(arrayBuffer);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  const base64 = window.btoa(binary);
  return base64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function base64Decode(base64UrlSafe: string) {
  // Pad the base64 string if needed
  const padding = "=".repeat((4 - (base64UrlSafe.length % 4)) % 4);
  const base64 = (base64UrlSafe + padding)
    .replace(/-/g, "+")
    .replace(/_/g, "/");

  // Decode the base64 string to binary data
  const binaryString = atob(base64);

  // Create an ArrayBuffer from the binary string
  const arrayBuffer = new ArrayBuffer(binaryString.length);
  const uint8Array = new Uint8Array(arrayBuffer);

  for (let i = 0; i < binaryString.length; i++) {
    uint8Array[i] = binaryString.charCodeAt(i);
  }

  return arrayBuffer;
}

// Where to go after signing in. A protected backend sends the user to the
// sign-in page with `?rd=<url>`, and going through /authorize then sets the
// session cookie on that backend's host before redirecting to `rd`. `rd` can
// also be a path on the admin host. Either way, the server validates it.
export function postSigninRedirect(
  search: string = window.location.search,
): string {
  const rd = new URLSearchParams(search).get("rd");
  if (!rd) {
    return "/";
  }
  if (rd.startsWith("/")) {
    return rd;
  }
  return "/authorize?rd=" + encodeURIComponent(rd);
}

// The `?rd=` query string of the current page, to carry it over to another
// page in the sign-in flow. Empty if there is none.
export function rdQuery(search: string = window.location.search): string {
  const rd = new URLSearchParams(search).get("rd");
  return rd ? "?rd=" + encodeURIComponent(rd) : "";
}
