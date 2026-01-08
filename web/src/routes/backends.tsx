import {
  IconPencil,
  IconPlus,
  IconStack2,
  IconStack2Filled,
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
import { ApiListBackendsResponse } from "../api/api_types";

export async function BackendlistLoader(api: ApiService) {
  return await api.ListBackends();
}

export async function BackendlistAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const fqdn = formData.get("fqdn") as string;
  if (!fqdn) {
    throw new Response("Missing fqdn", { status: 400 });
  }
  await api.DeleteBackend(fqdn);
  return redirect("/backends");
}

export default function Backends() {
  const loader = useLoaderData() as ApiListBackendsResponse;
  const backends = loader.backends;
  return (
    <>
      <div>
        <Link
          to="/backends/new"
          className="inline-flex items-center justify-center h-10 gap-2 px-5 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-none whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700 disabled:cursor-not-allowed disabled:border-emerald-300 disabled:text-emerald-300 disabled:shadow-none"
        >
          <span className="order-2">New backend</span>
          <span className="relative only:-mx-4">
            <IconPlus size={24} />
          </span>
        </Link>

        {backends.length > 0 ? (
          <ul className="divide-y divide-slate-100 max-w-xl mt-4">
            {backends.map((b) => {
              const editUrl = "/backends/edit/" + b.fqdn;
              return (
                <li key={b.fqdn} className="flex items-center gap-4 px-4 py-3">
                  <div className="self-start">
                    <a
                      href="#"
                      className="relative inline-flex h-8 w-8 items-center justify-center rounded-full text-white"
                    >
                      {b.accessLevel === "PUBLIC" ? (
                        <IconStack2Filled
                          className="text-orange-500"
                          size={24}
                        />
                      ) : (
                        <IconStack2 className="text-emerald-500" size={24} />
                      )}
                    </a>
                  </div>

                  <div className="flex min-h-[2rem] flex-1 flex-col items-start justify-center gap-0 overflow-hidden">
                    <h4 className="w-full truncate text-base text-slate-700">
                      {b.fqdn}
                    </h4>
                    <p className="w-full truncate text-sm text-slate-500">
                      {b.upstreamUrl}
                    </p>
                  </div>

                  <div>
                    <Link
                      to={editUrl}
                      className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-emerald-50 hover:text-emerald-600 focus:bg-emerald-100 focus:text-emerald-700 focus-visible:outline-none disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                    >
                      <span className="relative only:-mx-5">
                        <span className="sr-only">Edit backend</span>
                        <IconPencil />
                      </span>
                    </Link>
                    <DialogTrigger>
                      <Button className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-red-50 hover:text-red-600 focus:bg-red-100 focus:text-red-700 focus-visible:outline-none disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent">
                        <span className="relative only:-mx-5">
                          <span className="sr-only">Delete backend</span>
                          <IconX />
                        </span>
                      </Button>
                      <ModalOverlay
                        className={
                          "fixed top-0 left-0 w-full h-[100dvh] bg-black/50 flex items-center justify-center"
                        }
                      >
                        <Modal className={"bg-white p-4 rounded-md"}>
                          <Dialog role="alertdialog" className={"outline-none"}>
                            {({ close }) => (
                              <>
                                <Heading
                                  slot="title"
                                  className={"text-lg font-semibold"}
                                >
                                  Remove backend
                                </Heading>
                                <p className="py-2">
                                  Are you sure you want to remove the backend{" "}
                                  {b.fqdn}?
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
                                      name="fqdn"
                                      value={b.fqdn}
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
          <div className="text-center py-16 max-w-xl mx-auto">
            <IconStack2 className="mx-auto text-slate-400" size={48} />
            <h3 className="mt-4 text-lg font-medium text-slate-800">
              No backends yet
            </h3>
            <p className="mt-2 text-sm text-slate-600">
              Get started by creating a new backend.
            </p>
          </div>
        )}
      </div>
    </>
  );
}
