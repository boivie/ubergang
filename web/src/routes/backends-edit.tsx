import {
  ActionFunctionArgs,
  Form,
  redirect,
  useLoaderData,
  useParams,
} from "react-router";
import { ApiService } from "../api/api_client";
import { ApiBackend } from "../api/api_types";
import { useMemo, useState } from "react";
import { IconChevronDown, IconPlus, IconX } from "@tabler/icons-react";
import { StyledComboBox, StyledItem } from "../components/StyledComboBox";

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
  const [upstreamUrl, setUpstreamUrl] = useState(backend.upstreamUrl);
  const [accessLevel, setAccessLevel] = useState(
    backend.accessLevel || "NORMAL",
  );
  const [jsScript, setJsScript] = useState(backend.jsScript || "");
  const [isScriptExpanded, setIsScriptExpanded] = useState(!!backend.jsScript);

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

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">Edit {fqdn}</h1>
      <p className="mt-2 text-slate-600">Update backend configuration.</p>
      <Form className="mt-8 space-y-6" method="post">
        <input type="hidden" name="fqdn" value={fqdn} />
        <input type="hidden" name="headers" value={headersStr} />
        <input type="hidden" name="accessLevel" value={accessLevel} />
        <input type="hidden" name="jsScript" value={jsScript} />

        <div>
          <label
            htmlFor="upstream"
            className="block text-sm font-medium text-slate-700"
          >
            Upstream URL
          </label>
          <div className="mt-1">
            <input
              id="upstream"
              type="text"
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              name="upstream"
              value={upstreamUrl}
              onChange={(v) => setUpstreamUrl(v.target.value)}
            />
          </div>
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
