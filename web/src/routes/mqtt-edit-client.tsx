import {
  ActionFunctionArgs,
  Form,
  redirect,
  useLoaderData,
  useParams,
} from "react-router";
import { ApiService } from "../api/api_client";
import { ApiMqttClient } from "../api/api_types";
import { useMemo, useState } from "react";
import { IconPlus, IconRefresh, IconX } from "@tabler/icons-react";

export async function EditMqttClientLoader(api: ApiService, id: string) {
  return await api.GetMqttClient(id);
}

export async function EditMqttClientAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const id = payload.id as string;
  const profile_id = payload.profile_id as string;
  const password = (payload.password as string) || undefined;
  const valuesStr = payload.values as string;
  const values = valuesStr
    .split("\n")
    .filter((line) => line.includes("="))
    .reduce(
      (acc, line) => {
        const [key, ...valueParts] = line.split("=");
        if (key.trim()) {
          acc[key.trim()] = valueParts.join("=").trim();
        }
        return acc;
      },
      {} as { [key: string]: string },
    );

  await api.UpdateMqttClient(id, {
    id,
    profile_id,
    password,
    values,
  });
  return redirect("/mqtt/");
}

type EditableValue = { id: number; key: string; value: string };

export default function MqttEditClient() {
  const { id } = useParams<{ id: string }>();
  const client = useLoaderData() as ApiMqttClient;
  const [profileId, setProfileId] = useState(client.profile_id);
  const [clientId, setClientId] = useState(client.id);
  const [password, setPassword] = useState(client.password || "");

  const [values, setValues] = useState<EditableValue[]>(() =>
    Object.entries(client.values || {}).map(([key, value], index) => ({
      id: index,
      key,
      value,
    })),
  );

  const valuesStr = useMemo(
    () => values.map((v) => `${v.key}=${v.value}`).join("\n"),
    [values],
  );

  const handleValueChange = (
    id: number,
    field: "key" | "value",
    value: string,
  ) => {
    setValues(values.map((v) => (v.id === id ? { ...v, [field]: value } : v)));
  };

  const addValue = () => {
    setValues([...values, { id: Date.now(), key: "", value: "" }]);
  };

  const removeValue = (id: number) => {
    setValues(values.filter((v) => v.id !== id));
  };

  const generatePassword = () => {
    const chars =
      "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
    let pass = "";
    for (let i = 0; i < 16; i++) {
      pass += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setPassword(pass);
  };

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">Edit {id}</h1>
      <p className="mt-2 text-slate-600">Update MQTT client configuration.</p>
      <Form className="mt-8 space-y-6" method="post">
        <input type="hidden" name="id" value={id} />
        <input type="hidden" name="values" value={valuesStr} />

        <div>
          <label
            htmlFor="id"
            className="block text-sm font-medium text-slate-700"
          >
            MQTT Client ID
          </label>
          <div className="mt-1">
            <input
              id="id"
              name="id"
              type="text"
              required
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              value={clientId}
              onChange={(e) => setClientId(e.target.value)}
            />
          </div>
          <p className="mt-1 text-xs text-slate-500">
            The client ID used for MQTT connections.
          </p>
        </div>

        <div>
          <label
            htmlFor="profile_id"
            className="block text-sm font-medium text-slate-700"
          >
            Profile ID
          </label>
          <div className="mt-1">
            <input
              id="profile_id"
              name="profile_id"
              type="text"
              required
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              value={profileId}
              onChange={(e) => setProfileId(e.target.value)}
            />
          </div>
          <p className="mt-1 text-xs text-slate-500">
            The MQTT profile this client belongs to.
          </p>
        </div>

        <div>
          <label
            htmlFor="password"
            className="block text-sm font-medium text-slate-700"
          >
            Password
          </label>
          <div className="flex items-center mt-1 space-x-2">
            <input
              id="password"
              name="password"
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
            />
            <button
              type="button"
              onClick={generatePassword}
              className="p-2 inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-slate-100 hover:text-slate-600 focus:bg-slate-200 focus:text-slate-700 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
            >
              <IconRefresh size={20} />
            </button>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700">
            Custom Values
          </label>
          <div className="mt-2 space-y-2">
            {values.map((val) => (
              <div
                key={val.id}
                className="flex items-center space-x-2"
                data-testid={`value-row-${val.id}`}
              >
                <input
                  type="text"
                  placeholder="Key"
                  className="flex-1 px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
                  value={val.key}
                  onChange={(e) =>
                    handleValueChange(val.id, "key", e.target.value)
                  }
                />
                <input
                  type="text"
                  placeholder="Value"
                  className="flex-1 px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
                  value={val.value}
                  onChange={(e) =>
                    handleValueChange(val.id, "value", e.target.value)
                  }
                />
                <button
                  type="button"
                  aria-label={`Remove ${val.key} value`}
                  onClick={() => removeValue(val.id)}
                  className="p-2 inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-red-50 hover:text-red-600 focus:bg-red-100 focus:text-red-700 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                >
                  <IconX size={20} />
                </button>
              </div>
            ))}
          </div>
          <button
            type="button"
            onClick={addValue}
            className="inline-flex items-center justify-center h-10 gap-2 px-5 mt-2 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-hidden whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700 disabled:cursor-not-allowed disabled:border-emerald-300 disabled:text-emerald-300 disabled:shadow-none"
          >
            <IconPlus size={16} />
            Add Custom Value
          </button>
          <p className="mt-1 text-xs text-slate-500">
            Add custom key-value pairs for this client configuration.
          </p>
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
