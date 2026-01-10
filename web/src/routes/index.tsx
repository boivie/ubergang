import {
  IconChevronRight,
  IconDeviceDesktop,
  IconDeviceMobile,
  IconDeviceTablet,
  IconPencil,
  IconPlus,
  IconX,
} from "@tabler/icons-react";
import {
  ActionFunctionArgs,
  Form,
  Link,
  redirect,
  useLoaderData,
} from "react-router";
import {
  Button,
  Dialog,
  DialogTrigger,
  Heading,
  Modal,
  ModalOverlay,
} from "react-aria-components";
import { ApiService } from "../api/api_client";
import { relative_date } from "../lib/date_utils";
import { UAParser } from "ua-parser-js";
import { ApiUser } from "../api/api_types";

export async function IndexLoader(api: ApiService) {
  return await api.GetUser("me");
}

export async function IndexAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const id = formData.get("id") as string;
  if (!id) {
    throw new Response("Missing id", { status: 400 });
  }
  await api.DeleteCredential(id);
  return redirect("/");
}

function getDeviceIcon(userAgent: string) {
  const { device } = UAParser(userAgent);
  switch (device.type) {
    case "mobile":
      return <IconDeviceMobile />;
    case "tablet":
      return <IconDeviceTablet />;
    default:
      return <IconDeviceDesktop />;
  }
}

export default function Index() {
  const user = useLoaderData() as ApiUser;
  const now = new Date();
  return (
    <>
      <h1 className="text-2xl mb-3">Hello, {user.displayName}</h1>
      <h2 className="text-xl mb-2">Passkeys</h2>
      <div>
        <>
          <Link
            to="/enroll"
            className="inline-flex items-center justify-center h-10 gap-2 px-5 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-hidden whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700 disabled:cursor-not-allowed disabled:border-emerald-300 disabled:text-emerald-300 disabled:shadow-none"
          >
            <span className="order-2">Create a passkey</span>
            <span className="relative only:-mx-4">
              <IconPlus size={24} />
            </span>
          </Link>

          <ul className="divide-y divide-slate-100 max-w-xl">
            {user.credentials.map((e) => (
              <li key={e.id} className="flex items-center gap-4 px-4 py-3">
                <div className="self-start">
                  <a
                    href="#"
                    className="relative inline-flex h-8 w-8 items-center justify-center rounded-full text-white"
                  >
                    <img
                      src={`/passkey-image/${e.aaguid}`}
                      width={24}
                      height={24}
                    />
                  </a>
                </div>

                <div className="flex min-h-8 flex-1 flex-col items-start justify-center gap-0 overflow-hidden">
                  <h4 className="w-full truncate text-base text-slate-700">
                    {e.name}
                  </h4>
                  <p className="w-full truncate text-sm text-slate-500">
                    Added {relative_date(new Date(e.createdAt), now)}, last used{" "}
                    {relative_date(new Date(e.lastUsedAt), now)}
                  </p>
                </div>

                <div>
                  <Link
                    to={`/credentials/edit/${e.id}`}
                    className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-emerald-50 hover:text-emerald-600 focus:bg-emerald-100 focus:text-emerald-700 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                  >
                    <span className="relative only:-mx-5">
                      <span className="sr-only">Edit passkey</span>
                      <IconPencil />
                    </span>
                  </Link>
                  <DialogTrigger>
                    <Button className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-red-50 hover:text-red-600 focus:bg-red-100 focus:text-red-700 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent">
                      <span className="relative only:-mx-5">
                        <span className="sr-only">Delete passkey</span>
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
                              <Heading
                                slot="title"
                                className={"text-lg font-semibold"}
                              >
                                Remove passkey
                              </Heading>
                              <p className="py-2">
                                Are you sure you want to remove the passkey{" "}
                                {e.name}?
                              </p>
                              <div className="flex justify-end gap-2 pt-2">
                                <Button
                                  className="rounded-md bg-slate-200 px-3 py-1"
                                  onPress={close}
                                >
                                  Cancel
                                </Button>
                                <Form method="post">
                                  <input type="hidden" name="id" value={e.id} />
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
                </div>
              </li>
            ))}
          </ul>
        </>
      </div>
      <h2 className="text-xl mt-4 mb-2">Sessions</h2>
      <div>
        <ul className="divide-y divide-slate-100 max-w-xl">
          {user.sessions.map((e) => {
            const { browser, os } = UAParser(e.userAgent);
            return (
              <li key={e.id} className="flex items-center gap-4 px-4 py-3">
                <div className="self-start">{getDeviceIcon(e.userAgent)}</div>
                <div className="flex min-h-8 flex-1 flex-col items-start justify-center gap-0 overflow-hidden">
                  <h4 className="w-full truncate text-base text-slate-700">
                    {browser.name} on {os.name}
                    {e.id === user.currentSession?.id && (
                      <span className="ml-2 text-xs text-white bg-blue-500 rounded-full px-2 py-1">
                        Current
                      </span>
                    )}
                  </h4>
                  <p className="w-full truncate text-sm text-slate-500">
                    Last used {e.accessedAt} from {e.remoteAddr}
                  </p>
                </div>
                <div>
                  <Link
                    to={`/sessions/${e.id}`}
                    className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-emerald-50 hover:text-emerald-600 focus:bg-emerald-100 focus:text-emerald-700 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                  >
                    <span className="relative only:-mx-5">
                      <span className="sr-only">Session details</span>
                      <IconChevronRight />
                    </span>
                  </Link>
                </div>
              </li>
            );
          })}
        </ul>
      </div>
    </>
  );
}
