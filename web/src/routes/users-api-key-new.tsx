import { IconCheck, IconCopy } from "@tabler/icons-react";
import { useState } from "react";
import {
  ActionFunctionArgs,
  Form,
  Link,
  useActionData,
  useNavigation,
} from "react-router";
import { ApiService } from "../api/api_client";
import { ApiCreateApiKeyResponse, ApiKeyScope } from "../api/api_types";

export async function NewApiKeyAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
): Promise<ApiCreateApiKeyResponse> {
  const formData = await request.formData();
  const name = (formData.get("name") as string).trim();
  const scope = formData.get("scope") as ApiKeyScope;
  return await api.CreateApiKey({ name, scope });
}

const scopes: { value: ApiKeyScope; label: string; description: string }[] = [
  {
    value: "read",
    label: "Read",
    description: "List and view MQTT profiles and clients, without passwords.",
  },
  {
    value: "write",
    label: "Read and write",
    description:
      "Also create, change and delete MQTT profiles and clients, and read and set client passwords.",
  },
  {
    value: "admin",
    label: "Admin",
    description:
      "Also download database backups. A backup contains every secret on this server, so treat this key like full admin access.",
  },
];

function CreatedApiKey({ created }: { created: ApiCreateApiKeyResponse }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(created.token);
      setCopied(true);
    } catch {
      // Clipboard access can be denied; the token can still be selected.
    }
  };
  const example = `curl -H "Authorization: Bearer ${created.token}" \\\n  ${window.location.origin}/api/v1/mqtt/clients`;

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">API key created</h1>
      <p className="mt-2 text-slate-600">
        Copy the key for <strong>{created.apiKey.name}</strong> now. It won't be
        shown again.
      </p>

      <div className="mt-6 flex items-stretch gap-2">
        <input
          readOnly
          aria-label="API key"
          data-testid="api-key-token"
          value={created.token}
          onFocus={(e) => e.target.select()}
          className="block w-full min-w-0 px-3 py-2 font-mono text-sm border border-gray-300 rounded-md bg-slate-50"
        />
        <button
          type="button"
          onClick={copy}
          className="inline-flex items-center gap-1 px-3 text-sm font-medium border rounded-md border-emerald-500 text-emerald-600 hover:bg-emerald-50"
        >
          {copied ? <IconCheck size={18} /> : <IconCopy size={18} />}
          {copied ? "Copied" : "Copy"}
        </button>
      </div>

      <p className="mt-6 text-sm text-slate-600">Example:</p>
      <pre className="mt-1 p-3 overflow-x-auto text-xs rounded-md bg-slate-800 text-slate-100">
        {example}
      </pre>

      <div className="mt-8">
        <Link
          to="/users"
          className="flex justify-center w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700"
        >
          Done
        </Link>
      </div>
    </div>
  );
}

export default function NewApiKey() {
  const created = useActionData() as ApiCreateApiKeyResponse | undefined;
  const navigation = useNavigation();
  if (created) {
    return <CreatedApiKey created={created} />;
  }

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">New API key</h1>
      <p className="mt-2 text-slate-600">
        Create a key for the public API, e.g. for provisioning MQTT clients from
        a script.
      </p>
      <Form className="mt-8 space-y-6" method="post">
        <div>
          <label
            htmlFor="name"
            className="block text-sm font-medium text-slate-700"
          >
            Name
          </label>
          <div className="mt-1">
            <input
              id="name"
              name="name"
              type="text"
              required
              maxLength={100}
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              placeholder="e.g. Device provisioning"
            />
          </div>
        </div>

        <fieldset>
          <legend className="block text-sm font-medium text-slate-700">
            Access
          </legend>
          <div className="mt-2 space-y-3">
            {scopes.map((s) => (
              <label key={s.value} className="flex items-start gap-3">
                <input
                  type="radio"
                  name="scope"
                  value={s.value}
                  defaultChecked={s.value === "read"}
                  className="mt-1 accent-emerald-600"
                />
                <span>
                  <span className="block text-sm font-medium text-slate-700">
                    {s.label}
                  </span>
                  <span className="block text-sm text-slate-500">
                    {s.description}
                  </span>
                </span>
              </label>
            ))}
          </div>
        </fieldset>

        <div>
          <button
            type="submit"
            disabled={navigation.state !== "idle"}
            className="flex justify-center w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500 disabled:opacity-50"
          >
            Create API key
          </button>
        </div>
      </Form>
    </div>
  );
}
