import {
  IconKey,
  IconPencil,
  IconPlus,
  IconUser,
  IconUserStar,
  IconX,
} from "@tabler/icons-react";
import {
  Button,
  Dialog,
  DialogTrigger,
  Heading,
  Modal,
  ModalOverlay,
} from "react-aria-components";
import {
  ActionFunctionArgs,
  Form,
  Link,
  redirect,
  useLoaderData,
} from "react-router";
import { ApiService } from "../api/api_client";
import { ApiApiKey, ApiUser } from "../api/api_types";
import { relative_date } from "../lib/date_utils";

type UserlistLoaderData = {
  users: ApiUser[];
  apiKeys: ApiApiKey[];
};

export async function UserlistLoader(
  api: ApiService,
): Promise<UserlistLoaderData> {
  const [users, apiKeys] = await Promise.all([
    api.ListUsers(),
    api.ListApiKeys(),
  ]);
  return { users: users.users, apiKeys: apiKeys.apiKeys };
}

export async function UserlistAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const intent = formData.get("intent");
  if (intent === "deleteApiKey") {
    const apiKeyId = formData.get("apiKeyId") as string;
    if (!apiKeyId) {
      throw new Response("Missing apiKeyId", { status: 400 });
    }
    await api.DeleteApiKey(apiKeyId);
    return redirect("/users");
  }

  const userId = formData.get("userId") as string;
  if (!userId) {
    throw new Response("Missing userId", { status: 400 });
  }
  await api.DeleteUser(userId);
  return redirect("/users");
}

const newButtonClassName =
  "inline-flex items-center justify-center h-10 gap-2 px-5 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-hidden whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700 disabled:cursor-not-allowed disabled:border-emerald-300 disabled:text-emerald-300 disabled:shadow-none";

const iconButtonClassName =
  "inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent";

function ConfirmDeleteButton(props: {
  label: string;
  title: string;
  message: string;
  fields: Record<string, string>;
}) {
  return (
    <DialogTrigger>
      <Button
        aria-label={props.label}
        className={`${iconButtonClassName} hover:bg-red-50 hover:text-red-600 focus:bg-red-100 focus:text-red-700`}
      >
        <span className="relative only:-mx-5">
          <IconX />
        </span>
      </Button>
      <ModalOverlay
        className={
          "fixed top-0 left-0 w-full h-dvh bg-black/50 flex items-center justify-center"
        }
      >
        <Modal className={"bg-white p-4 rounded-md"}>
          <Dialog role="alertdialog" className={"outline-hidden"}>
            {({ close }) => (
              <>
                <Heading slot="title" className={"text-lg font-semibold"}>
                  {props.title}
                </Heading>
                <p className="py-2">{props.message}</p>
                <div className="flex justify-end gap-2 pt-2">
                  <Button
                    className="rounded-md bg-slate-200 px-3 py-1"
                    onPress={close}
                  >
                    Cancel
                  </Button>
                  <Form method="post">
                    {Object.entries(props.fields).map(([name, value]) => (
                      <input
                        key={name}
                        type="hidden"
                        name={name}
                        value={value}
                      />
                    ))}
                    <Button
                      type="submit"
                      className="rounded-md bg-red-500 px-3 py-1 text-white"
                    >
                      Remove
                    </Button>
                  </Form>
                </div>
              </>
            )}
          </Dialog>
        </Modal>
      </ModalOverlay>
    </DialogTrigger>
  );
}

function ScopeBadge({ scope }: { scope: ApiApiKey["scope"] }) {
  const className =
    scope === "read"
      ? "bg-sky-100 text-sky-800"
      : scope === "write"
        ? "bg-emerald-100 text-emerald-800"
        : "bg-amber-100 text-amber-800";
  const label = scope === "write" ? "read + write" : scope;
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${className}`}
    >
      {label}
    </span>
  );
}

function ApiKeyList({ apiKeys }: { apiKeys: ApiApiKey[] }) {
  const now = new Date();
  return (
    <section className="mt-10" aria-labelledby="api-keys-heading">
      <h2
        id="api-keys-heading"
        className="text-xl font-bold text-slate-800 mb-1"
      >
        API keys
      </h2>
      <p className="text-sm text-slate-500 mb-4 max-w-xl">
        API keys give access to the public API at <code>/api/v1/</code>. They
        aren't bound to a user.
      </p>
      <Link to="/users/api-keys/new" className={newButtonClassName}>
        <span className="order-2">New API key</span>
        <span className="relative only:-mx-4">
          <IconPlus size={24} />
        </span>
      </Link>

      {apiKeys.length === 0 ? (
        <p className="mt-4 px-4 text-sm text-slate-500">No API keys yet.</p>
      ) : (
        <ul className="divide-y divide-slate-100 max-w-xl mt-4">
          {apiKeys.map((k) => (
            <li
              key={k.id}
              className="flex items-center gap-4 px-4 py-3"
              data-testid={`api-key-row-${k.id}`}
            >
              <div className="self-start">
                <span className="relative inline-flex h-8 w-8 items-center justify-center rounded-full">
                  <IconKey color={"#64748b"} size={24} />
                </span>
              </div>

              <div className="flex min-h-8 flex-1 flex-col items-start justify-center gap-0 overflow-hidden">
                <div className="w-full flex items-center gap-2">
                  <h4 className="truncate text-base text-slate-700">
                    {k.name}
                  </h4>
                  <ScopeBadge scope={k.scope} />
                </div>
                <p className="w-full truncate text-sm text-slate-500">
                  Created {relative_date(new Date(k.createdAt), now)}
                  {k.createdBy && <> by {k.createdBy}</>}, last used{" "}
                  {k.lastUsedAt
                    ? relative_date(new Date(k.lastUsedAt), now)
                    : "never"}
                </p>
              </div>

              <div>
                <ConfirmDeleteButton
                  label={`Delete API key ${k.name}`}
                  title="Remove API key"
                  message={`Are you sure you want to remove the API key ${k.name}? Anything using it will stop working.`}
                  fields={{ intent: "deleteApiKey", apiKeyId: k.id }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export default function Users() {
  const { users, apiKeys } = useLoaderData() as UserlistLoaderData;
  return (
    <>
      <div>
        <Link to="/users/new" className={newButtonClassName}>
          <span className="order-2">New user</span>
          <span className="relative only:-mx-4">
            <IconPlus size={24} />
          </span>
        </Link>

        <ul className="divide-y divide-slate-100 max-w-xl mt-4">
          {users.map((u) => {
            const editUrl = "/users/edit/" + u.id;
            return (
              <li
                key={u.id}
                className="flex items-center gap-4 px-4 py-3"
                data-testid={`user-row-${u.id}`}
              >
                <div className="self-start">
                  <a
                    href="#"
                    className="relative inline-flex h-8 w-8 items-center justify-center rounded-full text-white"
                  >
                    {u.isAdmin ? (
                      <IconUserStar color={"#f59e0b"} size={24} />
                    ) : (
                      <IconUser color={"#10b981"} size={24} />
                    )}
                  </a>
                </div>

                <div className="flex min-h-8 flex-1 flex-col items-start justify-center gap-0 overflow-hidden">
                  <div className="w-full flex items-center gap-2">
                    <h4 className="truncate text-base text-slate-700">
                      {u.displayName}
                    </h4>
                    {u.isAdmin && (
                      <span className="inline-flex items-center rounded-full bg-amber-100 px-2 py-1 text-xs font-medium text-amber-800">
                        admin
                      </span>
                    )}
                  </div>
                  <p className="w-full truncate text-sm text-slate-500">
                    {u.email}
                  </p>
                </div>

                <div>
                  <Link
                    to={editUrl}
                    aria-label={`Edit user ${u.email}`}
                    className={`${iconButtonClassName} hover:bg-emerald-50 hover:text-emerald-600 focus:bg-emerald-100 focus:text-emerald-700`}
                  >
                    <span className="relative only:-mx-5">
                      <IconPencil />
                    </span>
                  </Link>
                  <ConfirmDeleteButton
                    label={`Delete user ${u.email}`}
                    title="Remove user"
                    message={`Are you sure you want to remove the user ${u.email}?`}
                    fields={{ userId: u.id }}
                  />
                </div>
              </li>
            );
          })}
        </ul>
      </div>

      <ApiKeyList apiKeys={apiKeys} />
    </>
  );
}
