import {
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
import { ApiListUsersResponse } from "../api/api_types";

export async function UserlistLoader(api: ApiService) {
  return await api.ListUsers();
}

export async function UserlistAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const userId = formData.get("userId") as string;
  if (!userId) {
    throw new Response("Missing userId", { status: 400 });
  }
  await api.DeleteUser(userId);
  return redirect("/users");
}

export default function Users() {
  const loader = useLoaderData() as ApiListUsersResponse;
  const users = loader.users;
  return (
    <>
      <div>
        <Link
          to="/users/new"
          className="inline-flex items-center justify-center h-10 gap-2 px-5 text-sm font-medium tracking-wide transition duration-300 border rounded-full focus-visible:outline-none whitespace-nowrap border-emerald-500 text-emerald-500 hover:border-emerald-600 hover:text-emerald-600 focus:border-emerald-700 focus:text-emerald-700 disabled:cursor-not-allowed disabled:border-emerald-300 disabled:text-emerald-300 disabled:shadow-none"
        >
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

                <div className="flex min-h-[2rem] flex-1 flex-col items-start justify-center gap-0 overflow-hidden">
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
                    className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-emerald-50 hover:text-emerald-600 focus:bg-emerald-100 focus:text-emerald-700 focus-visible:outline-none disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                  >
                    <span className="relative only:-mx-5">
                      <IconPencil />
                    </span>
                  </Link>
                  <DialogTrigger>
                    <Button
                      aria-label={`Delete user ${u.email}`}
                      className="inline-flex h-10 items-center justify-center gap-2 justify-self-center whitespace-nowrap rounded-full px-5 text-sm font-medium tracking-wide text-slate-500 transition duration-300 hover:bg-red-50 hover:text-red-600 focus:bg-red-100 focus:text-red-700 focus-visible:outline-none disabled:cursor-not-allowed disabled:text-emerald-300 disabled:shadow-none disabled:hover:bg-transparent"
                    >
                      <span className="relative only:-mx-5">
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
                                Remove user
                              </Heading>
                              <p className="py-2">
                                Are you sure you want to remove the user{" "}
                                {u.email}?
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
                                    name="userId"
                                    value={u.id}
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
      </div>
    </>
  );
}
