import {
  ActionFunctionArgs,
  Form,
  redirect,
  useLoaderData,
  useParams,
} from "react-router";
import { ApiService } from "../api/api_client";
import { ApiMqttProfile } from "../api/api_types";
import { useState } from "react";

export async function EditMqttProfileLoader(api: ApiService, id: string) {
  return await api.GetMqttProfile(id);
}

export async function EditMqttProfileAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const id = payload.id as string;
  const allow_publish = (payload.allow_publish as string)
    .split("\n")
    .filter((t) => t.trim());
  const allow_subscribe = (payload.allow_subscribe as string)
    .split("\n")
    .filter((t) => t.trim());

  await api.UpdateMqttProfile(id, {
    allow_publish,
    allow_subscribe,
  });
  return redirect("/mqtt/");
}

export default function MqttEditProfile() {
  const { id } = useParams<{ id: string }>();
  const profile = useLoaderData() as ApiMqttProfile;
  const [allowPublish, setAllowPublish] = useState(
    profile.allow_publish.join("\n"),
  );
  const [allowSubscribe, setAllowSubscribe] = useState(
    profile.allow_subscribe.join("\n"),
  );

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">Edit {id}</h1>
      <p className="mt-2 text-slate-600">Update MQTT profile configuration.</p>
      <Form className="mt-8 space-y-6" method="post">
        <input type="hidden" name="id" value={id} />

        <div>
          <label
            htmlFor="allow_publish"
            className="block text-sm font-medium text-slate-700"
          >
            Allowed Publish Topics
          </label>
          <div className="mt-1">
            <textarea
              id="allow_publish"
              name="allow_publish"
              rows={4}
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-sm appearance-none focus:outline-none focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              value={allowPublish}
              onChange={(e) => setAllowPublish(e.target.value)}
              placeholder="topic/+/publish&#10;sensor/#&#10;device/+/data"
            />
          </div>
          <p className="mt-1 text-xs text-slate-500">
            One topic pattern per line. Use + for single-level wildcards, # for
            multi-level.
          </p>
        </div>

        <div>
          <label
            htmlFor="allow_subscribe"
            className="block text-sm font-medium text-slate-700"
          >
            Allowed Subscribe Topics
          </label>
          <div className="mt-1">
            <textarea
              id="allow_subscribe"
              name="allow_subscribe"
              rows={4}
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-sm appearance-none focus:outline-none focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              value={allowSubscribe}
              onChange={(e) => setAllowSubscribe(e.target.value)}
              placeholder="topic/+/subscribe&#10;command/#&#10;status/+/updates"
            />
          </div>
          <p className="mt-1 text-xs text-slate-500">
            One topic pattern per line. Use + for single-level wildcards, # for
            multi-level.
          </p>
        </div>

        <div>
          <button
            type="submit"
            className="flex justify-center w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-sm bg-emerald-600 hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
          >
            Save Changes
          </button>
        </div>
      </Form>
    </div>
  );
}
