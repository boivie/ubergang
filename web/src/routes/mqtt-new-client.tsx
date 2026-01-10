import { useState } from "react";
import { ActionFunctionArgs, Form, redirect, useParams } from "react-router";
import { ApiService } from "../api/api_client";

export async function NewMqttClientAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const id = payload.id! as string;
  const profile_id = payload.profile_id! as string;

  await api.UpdateMqttClient(id, {
    profile_id,
  });
  return redirect("/mqtt/edit-client/" + id);
}

export default function MqttNewClient() {
  const { profileId } = useParams<{ profileId: string }>();
  const [clientId, setClientId] = useState("");

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">New MQTT Client</h1>
      <p className="mt-2 text-slate-600">
        Create a new MQTT client for profile {profileId}.
      </p>
      <Form className="mt-8 space-y-6" method="post">
        <input type="hidden" name="profile_id" value={profileId} />

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
              placeholder="mqtt-client-001"
            />
          </div>
          <p className="mt-1 text-xs text-slate-500">
            The client ID used for MQTT connections.
          </p>
        </div>

        <div>
          <button
            type="submit"
            className="flex justify-center w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
          >
            Create Client
          </button>
        </div>
      </Form>
    </div>
  );
}
