import { ActionFunctionArgs, Form, redirect } from "react-router";
import { ApiService } from "../api/api_client";

export async function NewBackendAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const fqdn = payload.fqdn! as string;

  await api.UpdateBackend(fqdn, {});
  return redirect("/backends/edit/" + fqdn);
}

export default function BackendsNew() {
  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">New Backend</h1>
      <p className="mt-2 text-slate-600">
        Create a new backend to proxy requests to.
      </p>
      <Form className="mt-8 space-y-6" method="post">
        <div>
          <label
            htmlFor="fqdn"
            className="block text-sm font-medium text-slate-700"
          >
            FQDN
          </label>
          <div className="mt-1">
            <input
              id="fqdn"
              name="fqdn"
              type="text"
              required
              autoFocus
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-sm appearance-none focus:outline-none focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              defaultValue={""}
              placeholder="backend.example.com"
            />
          </div>
        </div>

        <div>
          <button
            type="submit"
            className="flex justify-center w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-sm bg-emerald-600 hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
          >
            Create Backend
          </button>
        </div>
      </Form>
    </div>
  );
}
