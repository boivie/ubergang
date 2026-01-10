import {
  IconCheck,
  IconCopy,
  IconDeviceDesktop,
  IconDeviceMobile,
  IconDeviceTablet,
  IconPlus,
  IconX,
} from "@tabler/icons-react";
import { useEffect, useState } from "react";
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
  redirect,
  useLoaderData,
  useParams,
} from "react-router";
import { UAParser } from "ua-parser-js";
import { ApiService, useApiService } from "../api/api_client";
import { ApiUser, ApiBackend } from "../api/api_types";
import { StyledComboBox, StyledItem } from "../components/StyledComboBox";
import { relative_date } from "../lib/date_utils";

export async function EditUserLoader(api: ApiService, userId: string) {
  const [user, backends] = await Promise.all([
    api.GetUser(userId),
    api.ListBackends(),
  ]);
  return { user, backends };
}

export async function EditUserAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const userId = payload.userId as string;

  // Handle credential deletion
  if (payload.action === "deleteCredential") {
    const credentialId = payload.credentialId as string;
    if (!credentialId) {
      throw new Response("Missing credential ID", { status: 400 });
    }
    await api.DeleteCredential(credentialId);
    return redirect(`/users/edit/${userId}`);
  }

  // Handle session deletion
  if (payload.action === "deleteSession") {
    const sessionId = payload.sessionId as string;
    if (!sessionId) {
      throw new Response("Missing session ID", { status: 400 });
    }
    await api.DeleteSession(sessionId);
    return redirect(`/users/edit/${userId}`);
  }

  // Handle user update
  const email = payload.email as string;
  const displayName = payload.displayName as string;
  const admin = payload.admin === "on";
  const allowedHostsStr = payload.allowedHosts as string;
  const allowedHosts = allowedHostsStr
    .split("\n")
    .filter((h) => h.trim())
    .map((h) => h.trim())
    .sort();
  const req = { email, displayName, admin, allowedHosts };
  await api.UpdateUser(userId, req);
  return redirect("/users/");
}

export default function UsersEdit() {
  const { id } = useParams<{ id: string }>();
  const { user, backends } = useLoaderData() as {
    user: ApiUser;
    backends: { backends: ApiBackend[] };
  };
  const api = useApiService();
  const [email, setEmail] = useState(user.email);
  const [displayName, setDisplayName] = useState(user.displayName);
  const [admin, setAdmin] = useState(user.isAdmin);
  const [allowedHosts, setAllowedHosts] = useState<string[]>(
    [...(user.allowedHosts || [])].sort(),
  );
  const [allowedHostsStr, setAllowedHostsStr] = useState("");
  const [recoveryUrl, setRecoveryUrl] = useState("");
  const [copySuccess, setCopySuccess] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);

  useEffect(() => {
    setAllowedHostsStr(allowedHosts.join("\n"));
  }, [allowedHosts]);

  const handleHostChange = (index: number, value: string) => {
    setAllowedHosts(allowedHosts.map((h, i) => (i === index ? value : h)));
  };

  const addHost = () => {
    setAllowedHosts([...allowedHosts, ""]);
  };

  const removeHost = (index: number) => {
    setAllowedHosts(allowedHosts.filter((_, i) => i !== index));
  };

  const generateRecoveryUrl = async () => {
    if (!id) return;

    setIsGenerating(true);
    try {
      const response = await api.RecoverUser(id);
      setRecoveryUrl(response.recoveryUrl);
    } catch (error) {
      console.error("Failed to generate recovery URL:", error);
      alert("Failed to generate recovery URL. Please try again.");
    } finally {
      setIsGenerating(false);
    }
  };

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(recoveryUrl);
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    } catch (error) {
      console.error("Failed to copy to clipboard:", error);
      alert("Failed to copy to clipboard");
    }
  };

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

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">
        Edit {user.displayName}
      </h1>
      <p className="mt-2 text-slate-600">Update user configuration.</p>
      <Form className="mt-8 space-y-6" method="post">
        <input type="hidden" name="userId" value={id} />
        <input type="hidden" name="allowedHosts" value={allowedHostsStr} />

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
              type="email"
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              name="email"
              value={email}
              onChange={(v) => setEmail(v.target.value)}
            />
          </div>
        </div>

        <div>
          <label
            htmlFor="displayName"
            className="block text-sm font-medium text-slate-700"
          >
            Display Name
          </label>
          <div className="mt-1">
            <input
              id="displayName"
              type="text"
              className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
              name="displayName"
              value={displayName}
              onChange={(v) => setDisplayName(v.target.value)}
            />
          </div>
        </div>

        <div>
          <label className="flex items-center">
            <input
              type="checkbox"
              name="admin"
              checked={admin}
              onChange={(e) => setAdmin(e.target.checked)}
              className="h-4 w-4 text-emerald-600 focus:ring-emerald-500 border-gray-300 rounded-sm"
            />
            <span className="ml-2 text-sm font-medium text-slate-700">
              Administrator
            </span>
          </label>
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700">
            Allowed Hosts
          </label>
          <div className="mt-2 space-y-2">
            {allowedHosts.map((host, index) => (
              <div
                key={index}
                className="flex items-center space-x-2"
                data-testid={`allowed-host-row-${index}`}
              >
                <StyledComboBox
                  aria-label="Allowed host"
                  defaultItems={backends.backends.map((b: ApiBackend) => ({
                    id: b.fqdn,
                    name: b.fqdn,
                  }))}
                  inputValue={host}
                  allowsCustomValue={true}
                  onInputChange={(value) => handleHostChange(index, value)}
                >
                  {(item) => <StyledItem id={item.id}>{item.name}</StyledItem>}
                </StyledComboBox>
                <button
                  type="button"
                  onClick={() => removeHost(index)}
                  aria-label={`Remove host ${host}`}
                  className="p-2 inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-red-50 hover:text-red-600 focus:bg-red-100 focus:text-red-700 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                >
                  <IconX size={20} />
                </button>
              </div>
            ))}
          </div>
          <button
            type="button"
            onClick={addHost}
            className="inline-flex items-center justify-center h-10 gap-2 px-5 mt-2 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-hidden whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700 disabled:cursor-not-allowed disabled:border-emerald-300 disabled:text-emerald-300 disabled:shadow-none"
          >
            <IconPlus size={16} />
            Add Host
          </button>
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

      <div className="mt-8">
        <h2 className="text-xl font-bold text-slate-800 mb-4">
          Recovery Access
        </h2>
        <div className="bg-slate-50 p-4 rounded-md">
          <p className="text-sm text-slate-600 mb-4">
            Generate a recovery signin URL that can be sent to the user if they
            can't log in normally.
          </p>

          {!recoveryUrl ? (
            <button
              type="button"
              onClick={generateRecoveryUrl}
              disabled={isGenerating}
              className="inline-flex items-center justify-center h-10 gap-2 px-5 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-hidden whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700 disabled:cursor-not-allowed disabled:border-emerald-300 disabled:text-emerald-300 disabled:shadow-none"
            >
              {isGenerating ? "Generating..." : "Generate Recovery URL"}
            </button>
          ) : (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={recoveryUrl}
                  readOnly
                  className="block w-full px-3 py-2 text-sm bg-white border border-gray-300 rounded-md shadow-xs focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500"
                />
                <button
                  type="button"
                  onClick={copyToClipboard}
                  className="inline-flex items-center justify-center h-10 gap-2 px-3 text-sm font-medium tracking-wide transition duration-300 border rounded-md focus-visible:outline-hidden whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700"
                >
                  {copySuccess ? (
                    <IconCheck size={16} />
                  ) : (
                    <IconCopy size={16} />
                  )}
                  {copySuccess ? "Copied!" : "Copy"}
                </button>
              </div>
              <button
                type="button"
                onClick={() => setRecoveryUrl("")}
                className="text-sm text-slate-500 hover:text-slate-700"
              >
                Generate new URL
              </button>
            </div>
          )}
        </div>
      </div>

      <div className="mt-8">
        <h2 className="text-xl font-bold text-slate-800 mb-4">Passkeys</h2>
        {user.credentials && user.credentials.length > 0 ? (
          <ul className="divide-y divide-slate-100">
            {user.credentials.map((credential) => (
              <li
                key={credential.id}
                className="flex items-center gap-4 px-4 py-3"
                data-testid={`credential-row-${credential.id}`}
              >
                <div className="self-start">
                  <a
                    href="#"
                    className="relative inline-flex h-8 w-8 items-center justify-center rounded-full text-white"
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      shape-rendering="geometricPrecision"
                      text-rendering="geometricPrecision"
                      image-rendering="optimizeQuality"
                      fill-rule="evenodd"
                      clip-rule="evenodd"
                      viewBox="0 0 512 447.83"
                    >
                      <path d="M6.206 425.469A6.202 6.202 0 010 419.263c0-1.771.235-3.517.663-5.245 9.95-78.847 57.22-96.006 100.964-107.256 21.008-5.412 62.901-26.489 57.822-53.668-10.596-9.819-21.113-23.39-22.946-43.63l-1.274.026c-2.932-.044-5.778-.716-8.422-2.217-5.848-3.325-9.051-9.697-10.596-16.958-3.238-22.186-4.058-33.515 7.768-38.464l.096-.035c-1.467-27.37 3.159-67.64-24.944-76.141C154.622 7.1 218.597-30.203 266.627 30.805c53.519 2.801 77.389 78.611 44.154 121.046h-1.405c11.826 4.949 10.045 17.674 7.767 38.464-1.544 7.261-4.747 13.633-10.595 16.958-2.645 1.501-5.481 2.173-8.422 2.217l-1.275-.026c-1.833 20.24-12.376 33.811-22.972 43.63-4.459 23.871 27.275 43.011 49.33 50.997a139.442 139.442 0 004.268 5.525c-7.2 9.941-8.771 22.797-4.564 34.038-17.002 8.763-24.255 29.334-16.479 46.86-11.966 7.55-18.258 21.305-16.609 34.955H6.206zm419.058-105.362a86.778 86.778 0 0019.446-1.641c29.36-5.822 53.152-27.467 62.657-55.823 3.596-10.544 5.071-21.663 4.521-33.341-1.179-23.46-12.193-46.633-29.631-62.369-17.107-15.361-38.342-23.574-61.383-23.042-22.841.507-45.359 11.216-60.187 28.593-14.977 17.551-22.003 39.144-20.921 62.133.585 11.661 3.482 23.016 8.737 34.074 7.741 16.303 19.934 29.735 35.391 39.048l-22.457 20.711 13.232 27.86-29.413 13.973 13.991 29.456-26.769 12.717 16.81 35.374 35.601-16.915 40.375-110.808zm15.239-129.364c13.685 4.869 20.824 19.908 15.954 33.593-4.87 13.685-19.908 20.833-33.593 15.963-13.685-4.87-20.834-19.917-15.963-33.602 4.87-13.685 19.917-20.825 33.602-15.954z" />
                    </svg>
                  </a>
                </div>

                <div className="flex min-h-8 flex-1 flex-col items-start justify-center gap-0 overflow-hidden">
                  <h4 className="w-full truncate text-base text-slate-700">
                    {credential.name}
                  </h4>
                  <p className="w-full truncate text-sm text-slate-500">
                    Added{" "}
                    {relative_date(new Date(credential.createdAt), new Date())},
                    last used{" "}
                    {relative_date(new Date(credential.lastUsedAt), new Date())}
                  </p>
                </div>

                <div>
                  <DialogTrigger>
                    <Button
                      aria-label={`Delete passkey ${credential.name}`}
                      className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-red-50 hover:text-red-600 focus:bg-red-100 focus:text-red-700 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                    >
                      <span className="relative only:-mx-5">
                        <IconX />
                      </span>
                    </Button>
                    <ModalOverlay className="fixed top-0 left-0 w-full h-dvh bg-black/50 flex items-center justify-center">
                      <Modal className="bg-white p-4 rounded-md">
                        <Dialog role="alertdialog" className="outline-hidden">
                          {({ close }) => (
                            <>
                              <Heading
                                slot="title"
                                className="text-lg font-semibold"
                              >
                                Remove passkey
                              </Heading>
                              <p className="py-2">
                                Are you sure you want to remove the passkey{" "}
                                {credential.name}?
                              </p>
                              <div className="flex justify-end gap-2 pt-2">
                                <Button
                                  className="rounded-md bg-slate-200 px-3 py-1"
                                  onPress={close}
                                >
                                  Cancel
                                </Button>
                                <Form method="post">
                                  <input
                                    type="hidden"
                                    name="action"
                                    value="deleteCredential"
                                  />
                                  <input
                                    type="hidden"
                                    name="userId"
                                    value={id}
                                  />
                                  <input
                                    type="hidden"
                                    name="credentialId"
                                    value={credential.id}
                                  />
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
        ) : (
          <p className="text-slate-500 px-4 py-3">No passkeys found.</p>
        )}
      </div>

      <div className="mt-8">
        <h2 className="text-xl font-bold text-slate-800 mb-4">Sessions</h2>
        {user.sessions && user.sessions.length > 0 ? (
          <ul className="divide-y divide-slate-100">
            {user.sessions.map((session) => {
              const { browser, os } = UAParser(session.userAgent);
              return (
                <li
                  key={session.id}
                  className="flex items-center gap-4 px-4 py-3"
                  data-testid={`session-row-${session.id}`}
                >
                  <div className="self-start">
                    {getDeviceIcon(session.userAgent)}
                  </div>
                  <div className="flex min-h-8 flex-1 flex-col items-start justify-center gap-0 overflow-hidden">
                    <h4 className="w-full truncate text-base text-slate-700">
                      {browser.name} on {os.name}
                      {session.id === user.currentSession?.id && (
                        <span className="ml-2 text-xs text-white bg-blue-500 rounded-full px-2 py-1">
                          Current
                        </span>
                      )}
                    </h4>
                    <p className="w-full truncate text-sm text-slate-500">
                      Last used {session.accessedAt} from {session.remoteAddr}
                    </p>
                  </div>
                  <div>
                    <DialogTrigger>
                      <Button
                        aria-label={`Delete session for ${browser.name} on ${os.name}`}
                        className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-red-50 hover:text-red-600 focus:bg-red-100 focus:text-red-700 focus-visible:outline-hidden disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                      >
                        <span className="relative only:-mx-5">
                          <IconX />
                        </span>
                      </Button>
                      <ModalOverlay className="fixed top-0 left-0 w-full h-dvh bg-black/50 flex items-center justify-center">
                        <Modal className="bg-white p-4 rounded-md">
                          <Dialog role="alertdialog" className="outline-hidden">
                            {({ close }) => (
                              <>
                                <Heading
                                  slot="title"
                                  className="text-lg font-semibold"
                                >
                                  Remove session
                                </Heading>
                                <p className="py-2">
                                  Are you sure you want to remove this session
                                  for {browser.name} on {os.name}?
                                </p>
                                <div className="flex justify-end gap-2 pt-2">
                                  <Button
                                    className="rounded-md bg-slate-200 px-3 py-1"
                                    onPress={close}
                                  >
                                    Cancel
                                  </Button>
                                  <Form method="post">
                                    <input
                                      type="hidden"
                                      name="action"
                                      value="deleteSession"
                                    />
                                    <input
                                      type="hidden"
                                      name="userId"
                                      value={id}
                                    />
                                    <input
                                      type="hidden"
                                      name="sessionId"
                                      value={session.id}
                                    />
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
              );
            })}
          </ul>
        ) : (
          <p className="text-slate-500 px-4 py-3">No sessions found.</p>
        )}
      </div>
    </div>
  );
}
