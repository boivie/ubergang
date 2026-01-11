import {
  ActionFunctionArgs,
  Form,
  redirect,
  useLoaderData,
  useParams,
} from "react-router";
import { ApiService } from "../api/api_client";
import { ApiBackend, ApiValidateBackendResponse } from "../api/api_types";
import { useMemo, useState, useEffect, useRef } from "react";
import {
  IconChevronDown,
  IconPlus,
  IconX,
  IconCheck,
  IconAlertCircle,
  IconLoader2,
} from "@tabler/icons-react";
import { StyledComboBox, StyledItem } from "../components/StyledComboBox";
import { useApiService } from "../api/api_client";

export async function EditBackendLoader(api: ApiService, fqdn: string) {
  return await api.GetBackend(fqdn);
}

export async function EditBackendAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const fqdn = payload.fqdn as string;
  const upstreamUrl = payload.upstream as string;
  const headersStr = payload.headers as string;
  const accessLevel = payload.accessLevel as string;
  const jsScript = payload.jsScript as string;
  const pinnedCertificateFingerprint =
    (payload.pinnedCertificateFingerprint as string) || "";
  const headers = headersStr
    .split("\n")
    .filter((h) => h.includes("="))
    .map((h) => {
      const [name, ...valueParts] = h.split("=");
      return { name, value: valueParts.join("=") };
    });

  await api.UpdateBackend(fqdn, {
    upstreamUrl,
    headers,
    accessLevel,
    jsScript,
    pinnedCertificateFingerprint,
  });
  return redirect("/backends/");
}

type EditableHeader = { id: number; name: string; value: string };

const commonHeaders = [
  "Host",
  "Scheme",
  "X-Forwarded-For",
  "X-Forwarded-Proto",
  "X-Forwarded-Host",
];

const commonHeaderValues: { [key: string]: string[] } = {
  Host: ["$upstream_host"],
  Scheme: ["http", "https"],
  "X-Forwarded-Proto": ["http", "https"],
  "X-Forwarded-Host": ["$remote_addr"],
  "X-Forwarded-For": ["$remote_addr"],
};

export default function BackendsEdit() {
  const { fqdn } = useParams<{ fqdn: string }>();
  const backend = useLoaderData() as ApiBackend;
  const api = useApiService();
  const [upstreamUrl, setUpstreamUrl] = useState(backend.upstreamUrl);
  const [accessLevel, setAccessLevel] = useState(
    backend.accessLevel || "NORMAL",
  );
  const [jsScript, setJsScript] = useState(backend.jsScript || "");
  const [isScriptExpanded, setIsScriptExpanded] = useState(!!backend.jsScript);
  const [pinnedCertificateFingerprint, setPinnedCertificateFingerprint] =
    useState(backend.pinnedCertificateFingerprint || "");
  const [isPinnedCertExpanded, setIsPinnedCertExpanded] = useState(
    !!backend.pinnedCertificateFingerprint,
  );
  const [isValidating, setIsValidating] = useState(false);
  const [validationResult, setValidationResult] =
    useState<ApiValidateBackendResponse | null>(null);
  const debounceTimerRef = useRef<number | null>(null);

  const [headers, setHeaders] = useState<EditableHeader[]>(() =>
    backend.headers.map((h, i) => ({ id: i, ...h })),
  );

  const headersStr = useMemo(
    () => headers.map((h) => `${h.name}=${h.value}`).join("\n"),
    [headers],
  );

  const handleHeaderChange = (
    id: number,
    field: "name" | "value",
    value: string,
  ) => {
    setHeaders(
      headers.map((h) => (h.id === id ? { ...h, [field]: value } : h)),
    );
  };

  const addHeader = () => {
    setHeaders([...headers, { id: Date.now(), name: "", value: "" }]);
  };

  const removeHeader = (id: number) => {
    setHeaders(headers.filter((h) => h.id !== id));
  };

  const handleValidate = async (url: string) => {
    if (!url || url.trim() === "") {
      setValidationResult(null);
      return;
    }

    setIsValidating(true);
    try {
      const result = await api.ValidateBackend(url);
      setValidationResult(result);
    } catch (error) {
      setValidationResult({
        reachable: false,
        tls: false,
        error: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      setIsValidating(false);
    }
  };

  const handleUpstreamUrlChange = (value: string) => {
    setUpstreamUrl(value);

    // Clear existing timer
    if (debounceTimerRef.current) {
      window.clearTimeout(debounceTimerRef.current);
    }

    // Set new timer for debounced validation
    debounceTimerRef.current = window.setTimeout(() => {
      handleValidate(value);
    }, 1000); // Wait 1 second after user stops typing
  };

  const handleUpstreamUrlBlur = () => {
    // Clear debounce timer if exists
    if (debounceTimerRef.current) {
      window.clearTimeout(debounceTimerRef.current);
      debounceTimerRef.current = null;
    }
    // Validate immediately on blur
    handleValidate(upstreamUrl);
  };

  // Validate on initial load
  useEffect(() => {
    handleValidate(upstreamUrl);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        window.clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">Edit {fqdn}</h1>
      <p className="mt-2 text-slate-600">Update backend configuration.</p>
      <Form className="mt-8 space-y-6" method="post">
        <input type="hidden" name="fqdn" value={fqdn} />
        <input type="hidden" name="headers" value={headersStr} />
        <input type="hidden" name="accessLevel" value={accessLevel} />
        <input type="hidden" name="jsScript" value={jsScript} />
        <input
          type="hidden"
          name="pinnedCertificateFingerprint"
          value={pinnedCertificateFingerprint}
        />

        <div>
          <label
            htmlFor="upstream"
            className="block text-sm font-medium text-slate-700"
          >
            Upstream URL
          </label>
          <div className="mt-1 relative">
            <input
              id="upstream"
              type="text"
              className={`block w-full px-3 py-2 pr-10 placeholder-gray-400 border rounded-md shadow-xs appearance-none focus:outline-hidden sm:text-sm ${
                validationResult === null
                  ? "border-gray-300 focus:ring-emerald-500 focus:border-emerald-500"
                  : validationResult.reachable
                    ? "border-emerald-500 focus:ring-emerald-500 focus:border-emerald-600"
                    : "border-red-500 focus:ring-red-500 focus:border-red-600"
              }`}
              name="upstream"
              value={upstreamUrl}
              onChange={(e) => handleUpstreamUrlChange(e.target.value)}
              onBlur={handleUpstreamUrlBlur}
            />
            {isValidating && (
              <div className="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
                <IconLoader2
                  size={20}
                  className="text-slate-400 animate-spin"
                />
              </div>
            )}
            {!isValidating && validationResult && (
              <div className="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
                {validationResult.reachable ? (
                  <IconCheck size={20} className="text-emerald-500" />
                ) : (
                  <IconAlertCircle size={20} className="text-red-500" />
                )}
              </div>
            )}
          </div>
          {validationResult && !validationResult.reachable && (
            <p className="mt-1 text-sm text-red-600">
              {validationResult.error || "Backend is not reachable"}
            </p>
          )}
          {validationResult && validationResult.reachable && (
            <div className="mt-2">
              <p className="text-xs text-emerald-700 flex items-center gap-1">
                <IconCheck size={14} />
                Backend is reachable via{" "}
                {validationResult.tls ? "HTTPS" : "HTTP"}
              </p>
              {validationResult.certificates &&
                validationResult.certificates.length > 0 && (
                  <details className="mt-2 text-xs">
                    <summary className="cursor-pointer text-slate-600 hover:text-slate-800 font-medium">
                      Certificate Details (
                      {validationResult.certificates.length}{" "}
                      {validationResult.certificates.length > 1
                        ? "certificates"
                        : "certificate"}
                      )
                    </summary>
                    <div className="mt-2 space-y-2">
                      {validationResult.certificates.map((cert, idx) => (
                        <div
                          key={idx}
                          className="bg-slate-50 rounded p-3 border border-slate-200"
                        >
                          <div className="flex justify-between items-start mb-2">
                            <p className="font-medium text-slate-700">
                              Certificate {idx + 1}
                            </p>
                            <button
                              type="button"
                              onClick={() => {
                                setPinnedCertificateFingerprint(
                                  cert.sha256Fingerprint,
                                );
                                setIsPinnedCertExpanded(true);
                              }}
                              className="text-[10px] px-2 py-1 bg-emerald-600 text-white rounded hover:bg-emerald-700 transition-colors whitespace-nowrap"
                              title="Pin this certificate"
                            >
                              Pin this cert
                            </button>
                          </div>
                          <div className="space-y-1 text-slate-600">
                            <p className="break-all">
                              <span className="font-medium">Subject:</span>{" "}
                              {cert.subject}
                            </p>
                            <p className="break-all">
                              <span className="font-medium">Issuer:</span>{" "}
                              {cert.issuer}
                            </p>
                            <p>
                              <span className="font-medium">Valid:</span>{" "}
                              {new Date(cert.notBefore).toLocaleDateString()} -{" "}
                              {new Date(cert.notAfter).toLocaleDateString()}
                            </p>
                            {cert.dnsNames && cert.dnsNames.length > 0 && (
                              <p>
                                <span className="font-medium">DNS Names:</span>{" "}
                                {cert.dnsNames.join(", ")}
                              </p>
                            )}
                            <p className="break-all">
                              <span className="font-medium">SHA256:</span>{" "}
                              <span className="font-mono text-[10px]">
                                {cert.sha256Fingerprint}
                              </span>
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </details>
                )}
            </div>
          )}
        </div>

        <div className="border border-slate-200 rounded-lg overflow-hidden">
          <button
            type="button"
            onClick={() => setIsPinnedCertExpanded(!isPinnedCertExpanded)}
            className="w-full px-4 py-3 flex items-center justify-between text-left bg-slate-50 hover:bg-slate-100 transition-colors"
          >
            <div className="flex-1 min-w-0">
              <span className="text-sm font-medium text-slate-700">
                Certificate Pinning
              </span>
              <p className="text-xs text-slate-500 mt-0.5">
                {pinnedCertificateFingerprint ? (
                  <span className="font-mono truncate block">
                    Pinned: {pinnedCertificateFingerprint}
                  </span>
                ) : (
                  "Optional: Pin a specific TLS certificate"
                )}
              </p>
            </div>
            <IconChevronDown
              size={20}
              className={`text-slate-500 transition-transform flex-shrink-0 ml-2 ${
                isPinnedCertExpanded ? "rotate-180" : ""
              }`}
            />
          </button>
          {isPinnedCertExpanded && (
            <div className="p-4 bg-white">
              <input
                id="pinnedCertificateFingerprint"
                type="text"
                className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs font-mono text-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500"
                value={pinnedCertificateFingerprint}
                onChange={(e) =>
                  setPinnedCertificateFingerprint(e.target.value.trim())
                }
                placeholder="SHA256 fingerprint (leave empty to use system root CAs)"
              />
              <div className="mt-2 text-xs text-slate-600 space-y-1">
                <p>
                  Pin a specific certificate by SHA256 fingerprint for
                  additional security. Only applies to HTTPS backends.
                </p>
                {pinnedCertificateFingerprint && (
                  <button
                    type="button"
                    onClick={() => setPinnedCertificateFingerprint("")}
                    className="text-orange-600 hover:text-orange-700 font-medium"
                  >
                    Clear pinned certificate
                  </button>
                )}
              </div>
            </div>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700 mb-2">
            Access Level
          </label>
          <label className="inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              className="sr-only peer"
              checked={accessLevel === "PUBLIC"}
              onChange={(e) =>
                setAccessLevel(e.target.checked ? "PUBLIC" : "NORMAL")
              }
            />
            <div className="relative w-11 h-6 bg-emerald-600 peer-focus:outline-hidden peer-focus:ring-4 peer-focus:ring-emerald-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:rtl:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
            <span className="ms-3 text-sm font-medium text-slate-700">
              {accessLevel === "PUBLIC" ? (
                <span className="text-orange-600">
                  Public - No authentication required
                </span>
              ) : (
                <span className="text-emerald-600">
                  Normal - Requires authentication
                </span>
              )}
            </span>
          </label>
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700">
            Headers
          </label>
          <div className="mt-2 space-y-2">
            {headers.map((header) => (
              <div
                key={header.id}
                className="flex items-center space-x-2"
                data-testid={`header-row-${header.id}`}
              >
                <StyledComboBox
                  aria-label="Header name"
                  defaultItems={commonHeaders.map((h) => ({ id: h, name: h }))}
                  inputValue={header.name}
                  allowsCustomValue={true}
                  onInputChange={(value) =>
                    handleHeaderChange(header.id, "name", value)
                  }
                >
                  {(item) => <StyledItem id={item.id}>{item.name}</StyledItem>}
                </StyledComboBox>
                <StyledComboBox
                  aria-label="Header value"
                  defaultItems={(commonHeaderValues[header.name] || []).map(
                    (v) => ({ id: v, name: v }),
                  )}
                  inputValue={header.value}
                  allowsCustomValue={true}
                  onInputChange={(value) =>
                    handleHeaderChange(header.id, "value", value)
                  }
                >
                  {(item) => <StyledItem id={item.id}>{item.name}</StyledItem>}
                </StyledComboBox>
                <button
                  type="button"
                  aria-label={`Remove ${header.name} header`}
                  onClick={() => removeHeader(header.id)}
                  className="p-2 inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-orange-50 hover:text-orange-600 focus:bg-orange-100 focus:text-orange-700 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                >
                  <IconX size={20} />
                </button>
              </div>
            ))}
          </div>
          <button
            type="button"
            onClick={addHeader}
            className="inline-flex items-center justify-center h-10 gap-2 px-5 mt-2 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-hidden whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700 disabled:cursor-not-allowed disabled:border-emerald-300 disabled:text-emerald-300 disabled:shadow-none"
          >
            <IconPlus size={16} />
            Add Header
          </button>
        </div>

        <div className="border border-slate-200 rounded-lg overflow-hidden">
          <button
            type="button"
            onClick={() => setIsScriptExpanded(!isScriptExpanded)}
            className="w-full px-4 py-3 flex items-center justify-between text-left bg-slate-50 hover:bg-slate-100 transition-colors"
          >
            <div>
              <span className="text-sm font-medium text-slate-700">
                JavaScript Handler
              </span>
              <p className="text-xs text-slate-500 mt-0.5">
                Optional script to process requests before proxying
              </p>
            </div>
            <IconChevronDown
              size={20}
              className={`text-slate-500 transition-transform ${
                isScriptExpanded ? "rotate-180" : ""
              }`}
            />
          </button>
          {isScriptExpanded && (
            <div className="p-4 bg-white">
              <textarea
                value={jsScript}
                onChange={(e) => setJsScript(e.target.value)}
                placeholder=""
                className="w-full h-64 px-3 py-2 font-mono text-sm border border-slate-300 rounded-md focus:outline-hidden focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 resize-y"
                spellCheck={false}
              />
              <p className="mt-2 text-xs text-slate-500">
                Write JavaScript code to process requests. The handler function
                receives a request object and should return a modified request
                or response.
              </p>
            </div>
          )}
        </div>

        <div>
          <button
            type="submit"
            className="flex justify-center w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
          >
            Save Changes
          </button>
        </div>
      </Form>
    </div>
  );
}
