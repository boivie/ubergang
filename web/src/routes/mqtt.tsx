import {
  IconBox,
  IconCloud,
  IconCloudLock,
  IconCloudOff,
  IconDownload,
  IconPencil,
  IconPlus,
  IconUpload,
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
import { ApiService, useApiService } from "../api/api_client";
import {
  ApiListMqttClientsResponse,
  ApiListMqttProfilesResponse,
  ApiMqttClient,
} from "../api/api_types";

export async function MqttListLoader(api: ApiService) {
  const [profiles, clients] = await Promise.all([
    api.ListMqttProfiles(),
    api.ListMqttClients(),
  ]);

  return { profiles, clients };
}

export async function MqttListAction(
  _api: ApiService,
  _args: ActionFunctionArgs,
) {
  return redirect("/mqtt");
}

export default function MqttListComponent() {
  const loader = useLoaderData() as {
    profiles: ApiListMqttProfilesResponse;
    clients: ApiListMqttClientsResponse;
  };
  const profiles = loader.profiles.mqtt_profiles;
  const clients = loader.clients.mqtt_clients;
  const api = useApiService();

  profiles.sort((a, b) => a.id.localeCompare(b.id));
  clients.sort((a, b) => a.id.localeCompare(b.id));

  // Group clients by profiles:
  const clientsByProfile = new Map<string, ApiMqttClient[]>();
  for (const client of clients) {
    if (!clientsByProfile.has(client.profile_id)) {
      clientsByProfile.set(client.profile_id, []);
    }
    clientsByProfile.get(client.profile_id)?.push(client);
  }

  const handleExport = async () => {
    try {
      const blob = await api.ExportMqtt();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "mqtt-config.yaml";
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error("Failed to export MQTT configuration:", error);
      alert("Failed to export MQTT configuration");
    }
  };

  return (
    <>
      <div>
        <div className="flex flex-wrap gap-3 mb-4">
          <Link
            to="/mqtt/new-profile"
            className="inline-flex items-center justify-center h-10 gap-2 px-5 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-none whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700 disabled:cursor-not-allowed disabled:border-emerald-300 disabled:text-emerald-300 disabled:shadow-none"
          >
            <span className="order-2">New profile</span>
            <span className="relative only:-mx-4">
              <IconPlus size={24} />
            </span>
          </Link>
          <Link
            to="/mqtt/import"
            className="inline-flex items-center justify-center h-10 gap-2 px-5 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-none whitespace-nowrap border-blue-500 text-blue-500 hover:border-blue-600 hover:text-blue-600 focus:border-blue-700 focus:text-blue-700 disabled:cursor-not-allowed disabled:border-blue-300 disabled:text-blue-300 disabled:shadow-none"
          >
            <span className="order-2">Import</span>
            <span className="relative only:-mx-4">
              <IconUpload size={24} />
            </span>
          </Link>
          <button
            onClick={handleExport}
            disabled={profiles.length === 0 && clients.length === 0}
            className="inline-flex items-center justify-center h-10 gap-2 px-5 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-none whitespace-nowrap border-blue-500 text-blue-500 hover:border-blue-600 hover:text-blue-600 focus:border-blue-700 focus:text-blue-700 disabled:cursor-not-allowed disabled:border-slate-300 disabled:text-slate-300 disabled:shadow-none"
          >
            <span className="order-2">Export</span>
            <span className="relative only:-mx-4">
              <IconDownload size={24} />
            </span>
          </button>
        </div>
        {profiles.length > 0 ? (
          <div className="mt-4 space-y-6 max-w-xl">
            {profiles.map((profile) => {
              const profileClients = clientsByProfile.get(profile.id) || [];
              const editProfileUrl = "/mqtt/edit-profile/" + profile.id;

              return (
                <div
                  key={profile.id}
                  className="border border-slate-200 rounded-lg overflow-hidden"
                >
                  <div className="bg-slate-50 px-4 py-3 border-b border-slate-200">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div>
                          <h3 className="text-lg font-medium text-slate-700">
                            {profile.id}
                          </h3>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Link
                          to={editProfileUrl}
                          className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-emerald-50 hover:text-emerald-600 focus:bg-emerald-100 focus:text-emerald-700 focus-visible:outline-none disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                        >
                          <span className="relative only:-mx-5">
                            <span className="sr-only">Edit profile</span>
                            <IconPencil />
                          </span>
                        </Link>
                        {profileClients.length === 0 && (
                          <DialogTrigger>
                            <Button className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-red-50 hover:text-red-600 focus:bg-red-100 focus:text-red-700 focus-visible:outline-none disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent">
                              <span className="relative only:-mx-5">
                                <span className="sr-only">Delete profile</span>
                                <IconX />
                              </span>
                            </Button>
                            <ModalOverlay
                              className={
                                "fixed top-0 left-0 w-full h-[100dvh] bg-black/50 flex items-center justify-center"
                              }
                            >
                              <Modal className={"bg-white p-4 rounded-md"}>
                                <Dialog
                                  role="alertdialog"
                                  className={"outline-none"}
                                >
                                  {({ close }) => (
                                    <>
                                      <Heading
                                        slot="title"
                                        className={"text-lg font-semibold"}
                                      >
                                        Remove MQTT profile
                                      </Heading>
                                      <p className="py-2">
                                        Are you sure you want to remove the MQTT
                                        profile {profile.id}?
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
                                            name="profile_id"
                                            value={profile.id}
                                          />
                                          <input
                                            type="hidden"
                                            name="action"
                                            value="delete-profile"
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
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="p-4">
                    <div className="flex items-center justify-between mb-3">
                      <span></span>
                      <Link
                        to={`/mqtt/new-client/${profile.id}`}
                        className="inline-flex items-center justify-center h-10 gap-2 px-5 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-none whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700 disabled:cursor-not-allowed disabled:border-emerald-300 disabled:text-emerald-300 disabled:shadow-none"
                      >
                        <span className="order-2">Add Client</span>
                        <span className="relative only:-mx-2">
                          <IconPlus size={16} />
                        </span>
                      </Link>
                    </div>

                    {profileClients.length > 0 ? (
                      <ul className="divide-y divide-slate-100">
                        {profileClients.map((client) => {
                          const editClientUrl =
                            "/mqtt/edit-client/" + client.id;
                          return (
                            <li
                              key={client.id}
                              className="flex items-center gap-4 py-3"
                            >
                              <div className="self-start">
                                <a
                                  href="#"
                                  className="relative inline-flex h-6 w-6 items-center justify-center rounded-full text-white"
                                >
                                  {!client.connected ? (
                                    <IconCloudOff
                                      className="text-red-500"
                                      size={20}
                                    />
                                  ) : client.connected.connectionType ===
                                    "mqtt-tls" ? (
                                    <IconCloudLock
                                      className="text-emerald-500"
                                      size={20}
                                    />
                                  ) : (
                                    <IconCloud
                                      className="text-emerald-500"
                                      size={20}
                                    />
                                  )}
                                </a>
                              </div>

                              <div className="flex min-h-[2rem] flex-1 flex-col items-start justify-center gap-0 overflow-hidden">
                                <h5 className="w-full truncate text-sm text-slate-700">
                                  {client.id}
                                </h5>
                                <p className="w-full truncate text-xs text-slate-500">
                                  {client.connected
                                    ? `Connected from ${
                                        client.connected.remoteAddr
                                      } (${
                                        client.connected.connectionType ===
                                        "mqtt-tls"
                                          ? "TLS"
                                          : "plain"
                                      })`
                                    : "Not connected"}
                                </p>
                              </div>

                              <div>
                                <Link
                                  to={editClientUrl}
                                  className="inline-flex h-8 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-4 text-xs font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-emerald-50 hover:text-emerald-600 focus:bg-emerald-100 focus:text-emerald-700 focus-visible:outline-none disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                                >
                                  <span className="relative only:-mx-4">
                                    <span className="sr-only">Edit client</span>
                                    <IconPencil size={20} />
                                  </span>
                                </Link>
                                <DialogTrigger>
                                  <Button className="inline-flex h-8 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-4 text-xs font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-red-50 hover:text-red-600 focus:bg-red-100 focus:text-red-700 focus-visible:outline-none disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent">
                                    <span className="relative only:-mx-4">
                                      <span className="sr-only">
                                        Delete client
                                      </span>
                                      <IconX size={20} />
                                    </span>
                                  </Button>
                                  <ModalOverlay
                                    className={
                                      "fixed top-0 left-0 w-full h-[100dvh] bg-black/50 flex items-center justify-center"
                                    }
                                  >
                                    <Modal
                                      className={"bg-white p-4 rounded-md"}
                                    >
                                      <Dialog
                                        role="alertdialog"
                                        className={"outline-none"}
                                      >
                                        {({ close }) => (
                                          <>
                                            <Heading
                                              slot="title"
                                              className={
                                                "text-lg font-semibold"
                                              }
                                            >
                                              Remove MQTT client
                                            </Heading>
                                            <p className="py-2">
                                              Are you sure you want to remove
                                              the MQTT client {client.id}?
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
                                                  name="client_id"
                                                  value={client.id}
                                                />
                                                <input
                                                  type="hidden"
                                                  name="action"
                                                  value="delete-client"
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
                      <div className="text-center">
                        <IconCloudOff
                          className="mx-auto text-slate-400"
                          size={48}
                        />
                        <h4 className="mt-2 text-sm font-medium text-slate-600">
                          No clients yet
                        </h4>
                        <p className="mt-1 text-xs text-slate-500">
                          Add a client to this profile to get started.
                        </p>
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="text-center py-16 max-w-xl mx-auto">
            <IconBox className="mx-auto text-slate-400" size={48} />
            <h3 className="mt-4 text-lg font-medium text-slate-800">
              No MQTT profiles yet
            </h3>
            <p className="mt-2 text-sm text-slate-600">
              Get started by creating a new MQTT profile.
            </p>
          </div>
        )}
      </div>
    </>
  );
}
