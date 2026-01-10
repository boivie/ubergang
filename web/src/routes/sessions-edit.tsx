import { IconLogout2 } from "@tabler/icons-react";
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
} from "react-router";
import { UAParser } from "ua-parser-js";
import { ApiService } from "../api/api_client";
import { ApiSession } from "../api/api_types";
import { relative_date } from "../lib/date_utils";

export async function SessionEditLoader(
  api: ApiService,
  id: string,
): Promise<ApiSession | undefined> {
  const user = await api.GetUser("me");
  return user.sessions.find((s) => s.id === id);
}

export async function SessionEditAction(
  api: ApiService,
  { request, params }: ActionFunctionArgs,
) {
  const formData = await request.formData();
  const intent = formData.get("intent");

  if (intent === "delete") {
    const id = params.id as string;
    await api.DeleteSession(id);
    return redirect("/");
  }

  return null;
}

export default function SessionsEdit() {
  const session = useLoaderData() as ApiSession;
  const { browser, os } = UAParser(session.userAgent);
  const now = new Date();

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">Session Details</h1>
      <div className="mt-8 bg-white shadow-md rounded-lg overflow-hidden">
        <div className="px-4 py-5 sm:px-6 flex justify-between items-center">
          <div>
            <h3 className="text-lg leading-6 font-medium text-gray-900">
              {browser.name} on {os.name}
            </h3>
            <p className="mt-1 max-w-2xl text-sm text-gray-500">
              Session ID: {session.id}
            </p>
          </div>
        </div>
        <div className="border-t border-gray-200">
          <dl>
            <div className="bg-gray-50 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">User Agent</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {session.userAgent}
              </dd>
            </div>
            <div className="bg-white px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">
                Remote Address
              </dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {session.remoteAddr}
              </dd>
            </div>
            <div className="bg-gray-50 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">Created At</dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                {new Date(session.createdAt).toLocaleString()} (
                {relative_date(new Date(session.createdAt), now)})
              </dd>
            </div>
            <div className="bg-white px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">
                Last Accessed At
              </dt>
              <dd className="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
                ...
              </dd>
            </div>
          </dl>
        </div>
      </div>
      <DialogTrigger>
        <Button className="mt-3 inline-flex items-center justify-center h-10 gap-2 px-5 text-sm font-medium tracking-wide text-white transition duration-300 rounded-sm focus-visible:outline-hidden whitespace-nowrap bg-red-500 hover:bg-red-600 focus:bg-red-700 disabled:cursor-not-allowed disabled:border-red-300 disabled:bg-red-300 disabled:shadow-none">
          <span className="order-2">Sign out</span>
          <span className="relative only:-mx-5">
            <IconLogout2 size={24} />
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
                    Sign out session
                  </Heading>
                  <p className="py-2">
                    Are you sure you want to sign out this session?
                  </p>
                  <div className="flex justify-end gap-2 pt-2">
                    <Button
                      className="rounded-md bg-slate-200 px-3 py-1"
                      onPress={close}
                    >
                      Cancel
                    </Button>
                    <Form method="post">
                      <input type="hidden" name="intent" value="delete" />
                      <Button
                        type="submit"
                        className="rounded-md bg-red-500 px-3 py-1 text-white"
                      >
                        Sign out
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
  );
}
