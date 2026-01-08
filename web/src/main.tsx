import React, { Suspense } from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider, createBrowserRouter } from "react-router";
import { ApiServiceContext, realApiService } from "./api/api_client.ts";
import { Home } from "./components/Home.tsx";
import "./index.css";
import { WebauthnServiceContext } from "./lib/webauthn.tsx";
import { realWebauthnService } from "./lib/webauthn-service.ts";
import EditBackendComponent, {
  EditBackendAction,
  EditBackendLoader,
} from "./routes/backends-edit.tsx";
import NewBackendComponent, {
  NewBackendAction,
} from "./routes/backends-new.tsx";
import NewUserComponent, { NewUserAction } from "./routes/users-new.tsx";
import BackendListComponent, {
  BackendlistAction,
  BackendlistLoader,
} from "./routes/backends.tsx";
import MqttListComponent, {
  MqttListAction,
  MqttListLoader,
} from "./routes/mqtt.tsx";
import NewMqttProfileComponent, {
  NewMqttProfileAction,
} from "./routes/mqtt-new-profile.tsx";
import EditMqttProfileComponent, {
  EditMqttProfileAction,
  EditMqttProfileLoader,
} from "./routes/mqtt-edit-profile.tsx";
import NewMqttClientComponent, {
  NewMqttClientAction,
} from "./routes/mqtt-new-client.tsx";
import EditMqttClientComponent, {
  EditMqttClientAction,
  EditMqttClientLoader,
} from "./routes/mqtt-edit-client.tsx";
import ImportMqttComponent, {
  ImportMqttAction,
} from "./routes/mqtt-import.tsx";
import ConfirmComponent from "./routes/confirm";
import ConfirmPinComponent, {
  ConfirmPinLoader,
} from "./routes/confirm-pin.tsx";
import EnrollComponent from "./routes/enroll";
import IndexComponent, { IndexAction, IndexLoader } from "./routes/index";
import SigninComponent from "./routes/signin";
import SigninTokenComponent, { SigninTokenLoader } from "./routes/signin-token";
import UsersListComponent, {
  UserlistAction,
  UserlistLoader,
} from "./routes/users.tsx";
import EditUserComponent, {
  EditUserAction,
  EditUserLoader,
} from "./routes/users-edit.tsx";
import CredentialEditComponent, {
  CredentialEditAction,
  CredentialEditLoader,
} from "./routes/credentials-edit.tsx";
import SessionEditComponent, {
  SessionEditAction,
  SessionEditLoader,
} from "./routes/sessions-edit.tsx";
import SetupComponent from "./routes/setup.tsx";

const api = realApiService;
const router = createBrowserRouter([
  {
    path: "/",
    element: <Home />,
    children: [
      {
        index: true,
        path: "/",
        loader: () => IndexLoader(api),
        action: (args) => IndexAction(api, args),
        element: <IndexComponent />,
        handle: { tabid: "profile" },
      },
      {
        path: "/credentials/edit/:id",
        element: <CredentialEditComponent />,
        loader: ({ params }) => CredentialEditLoader(api, params.id!),
        action: (args) => CredentialEditAction(api, args),
        handle: { tabid: "profile" },
      },
      {
        path: "/sessions/:id",
        element: <SessionEditComponent />,
        loader: ({ params }) => SessionEditLoader(api, params.id!),
        action: (args) => SessionEditAction(api, args),
        handle: { tabid: "profile" },
      },
      {
        path: "/backends/",
        loader: () => BackendlistLoader(api),
        action: (args) => BackendlistAction(api, args),
        element: <BackendListComponent />,
        handle: { tabid: "backends" },
      },
      {
        path: "/backends/new",
        action: (args) => NewBackendAction(api, args),
        element: <NewBackendComponent />,
        handle: { tabid: "backends" },
      },
      {
        path: "/backends/edit/:fqdn",
        element: <EditBackendComponent />,
        loader: ({ params }) => EditBackendLoader(api, params.fqdn!),
        action: (args) => EditBackendAction(api, args),
        handle: { tabid: "backends" },
      },
      {
        path: "/mqtt/",
        loader: () => MqttListLoader(api),
        action: (args) => MqttListAction(api, args),
        element: <MqttListComponent />,
        handle: { tabid: "mqtt" },
      },
      {
        path: "/mqtt/new-profile",
        action: (args) => NewMqttProfileAction(api, args),
        element: <NewMqttProfileComponent />,
        handle: { tabid: "mqtt" },
      },
      {
        path: "/mqtt/edit-profile/:id",
        element: <EditMqttProfileComponent />,
        loader: ({ params }) => EditMqttProfileLoader(api, params.id!),
        action: (args) => EditMqttProfileAction(api, args),
        handle: { tabid: "mqtt" },
      },
      {
        path: "/mqtt/new-client/:profileId",
        action: (args) => NewMqttClientAction(api, args),
        element: <NewMqttClientComponent />,
        handle: { tabid: "mqtt" },
      },
      {
        path: "/mqtt/edit-client/:id",
        element: <EditMqttClientComponent />,
        loader: ({ params }) => EditMqttClientLoader(api, params.id!),
        action: (args) => EditMqttClientAction(api, args),
        handle: { tabid: "mqtt" },
      },
      {
        path: "/mqtt/import",
        action: (args) => ImportMqttAction(api, args),
        element: <ImportMqttComponent />,
        handle: { tabid: "mqtt" },
      },
      {
        path: "/users/",
        loader: () => UserlistLoader(api),
        action: (args) => UserlistAction(api, args),
        element: <UsersListComponent />,
        handle: { tabid: "users" },
      },
      {
        path: "/users/new",
        action: (args) => NewUserAction(api, args),
        element: <NewUserComponent />,
        handle: { tabid: "users" },
      },
      {
        path: "/users/edit/:id",
        element: <EditUserComponent />,
        loader: ({ params }) => EditUserLoader(api, params.id!),
        action: (args) => EditUserAction(api, args),
        handle: { tabid: "users" },
      },
      {
        path: "/enroll/",
        element: <EnrollComponent />,
      },
    ],
  },
  {
    path: "/setup",
    element: <SetupComponent />,
  },
  {
    path: "/signin/",
    element: <SigninComponent />,
  },

  {
    path: "/signin/:token",
    loader: ({ params }) => SigninTokenLoader(api, params.token!),
    element: <SigninTokenComponent />,
  },
  {
    path: "/confirm/",
    element: <ConfirmComponent />,
  },
  {
    path: "/confirm/:pin",
    loader: ({ params }) => ConfirmPinLoader(api, params.pin!),
    element: <ConfirmPinComponent />,
  },
]);

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ApiServiceContext.Provider value={realApiService}>
      <WebauthnServiceContext.Provider value={realWebauthnService}>
        <Suspense fallback={<div>Loading...</div>}>
          <RouterProvider router={router} />
        </Suspense>
      </WebauthnServiceContext.Provider>
    </ApiServiceContext.Provider>
  </React.StrictMode>,
);
