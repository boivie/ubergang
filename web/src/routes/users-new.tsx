import { ActionFunctionArgs, Form, redirect } from "react-router";
import { ApiService } from "../api/api_client";

export async function NewUserAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const email = payload.email! as string;

  const response = await api.CreateUser({ email });
  return redirect("/users/edit/" + response.id);
}

export default function UsersNew() {
  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">New User</h1>
      <p className="mt-2 text-slate-600">Create a new user account.</p>
      <Form className="mt-8 space-y-6" method="post">
        <div>
          <label
            htmlFor="email"
            className="block text-sm font-medium text-slate-700"
          >
            Email
          </label>
          <div className="mt-1">
            <input
              id="email"
              name="email"
              type="email"
              required
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              defaultValue={""}
              placeholder="user@example.com"
            />
          </div>
        </div>

        <div>
          <button
            type="submit"
            className="flex justify-center w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
          >
            Create User
          </button>
        </div>
      </Form>
    </div>
  );
}
