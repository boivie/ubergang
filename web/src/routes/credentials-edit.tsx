import {
  ActionFunctionArgs,
  Form,
  redirect,
  useLoaderData,
} from "react-router";
import { ApiService } from "../api/api_client";
import { ApiCredential } from "../api/api_types";
import { useState } from "react";

export async function CredentialEditLoader(
  api: ApiService,
  id: string,
): Promise<ApiCredential | undefined> {
  const user = await api.GetUser("me");
  return user.credentials.find((c) => c.id === id);
}

export async function CredentialEditAction(
  api: ApiService,
  { request, params }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const name = formData.get("name") as string;
  const id = params.id as string;
  await api.UpdateCredential(id, { name });
  return redirect("/");
}

export default function CredentialsEdit() {
  const credential = useLoaderData() as ApiCredential;
  const [name, setName] = useState(credential.name);

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">
        Edit {credential.name}
      </h1>
      <p className="mt-2 text-slate-600">Update credential configuration.</p>
      <Form className="mt-8 space-y-6" method="post">
        <input type="hidden" name="id" value={credential.id} />
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
              type="text"
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-sm appearance-none focus:outline-none focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              name="name"
              value={name}
              onChange={(v) => setName(v.target.value)}
            />
          </div>
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
