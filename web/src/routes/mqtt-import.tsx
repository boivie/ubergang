import { IconFileUpload, IconAlertCircle } from "@tabler/icons-react";
import {
  ActionFunctionArgs,
  Form,
  redirect,
  useActionData,
} from "react-router";
import { ApiService } from "../api/api_client";
import { useState } from "react";

export async function ImportMqttAction(
  api: ApiService,
  { request }: ActionFunctionArgs,
) {
  try {
    const formData = await request.formData();
    const file = formData.get("file") as File;

    if (!file) {
      return { error: "Please select a file to import" };
    }

    const result = await api.ImportMqtt(file);
    return redirect(
      `/mqtt?imported=${result.profiles_count}_profiles_${result.clients_count}_clients`,
    );
  } catch (error) {
    return {
      error:
        error instanceof Error
          ? error.message
          : "Failed to import MQTT configuration",
    };
  }
}

export default function MqttImportComponent() {
  const actionData = useActionData() as { error?: string } | undefined;
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [fileContent, setFileContent] = useState<string>("");

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setSelectedFile(file);
      // Read file content to display preview
      const reader = new FileReader();
      reader.onload = (event) => {
        setFileContent(event.target?.result as string);
      };
      reader.readAsText(file);
    }
  };

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold text-slate-800">
        Import MQTT Configuration
      </h1>
      <p className="mt-2 text-slate-600">
        Upload a YAML file to import MQTT profiles and clients.
      </p>

      {actionData?.error && (
        <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-md flex items-start gap-3">
          <IconAlertCircle className="text-red-600 shrink-0" size={20} />
          <div className="flex-1">
            <h3 className="text-sm font-medium text-red-800">Import Failed</h3>
            <p className="mt-1 text-sm text-red-700">{actionData.error}</p>
          </div>
        </div>
      )}

      <Form
        className="mt-8 space-y-6"
        method="post"
        encType="multipart/form-data"
      >
        <div>
          <label
            htmlFor="file"
            className="block text-sm font-medium text-slate-700"
          >
            YAML Configuration File
          </label>
          <div className="mt-1">
            <div className="flex items-center justify-center w-full">
              <label
                htmlFor="file"
                className="flex flex-col items-center justify-center w-full h-32 border-2 border-slate-300 border-dashed rounded-lg cursor-pointer bg-slate-50 hover:bg-slate-100"
              >
                <div className="flex flex-col items-center justify-center pt-5 pb-6">
                  <IconFileUpload className="mb-3 text-slate-400" size={40} />
                  <p className="mb-2 text-sm text-slate-500">
                    <span className="font-semibold">Click to upload</span> or
                    drag and drop
                  </p>
                  <p className="text-xs text-slate-500">YAML files only</p>
                </div>
                <input
                  id="file"
                  name="file"
                  type="file"
                  className="hidden"
                  accept=".yaml,.yml"
                  onChange={handleFileChange}
                  required
                />
              </label>
            </div>
          </div>
          {selectedFile && (
            <p className="mt-2 text-sm text-slate-600">
              Selected: <span className="font-medium">{selectedFile.name}</span>{" "}
              ({Math.round(selectedFile.size / 1024)} KB)
            </p>
          )}
        </div>

        {fileContent && (
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              File Preview
            </label>
            <div className="bg-slate-900 rounded-md p-4 overflow-auto max-h-96">
              <pre className="text-xs text-slate-100 font-mono">
                {fileContent}
              </pre>
            </div>
          </div>
        )}

        <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
          <h3 className="text-sm font-medium text-blue-800 mb-2">
            Import Behavior
          </h3>
          <ul className="text-sm text-blue-700 space-y-1 list-disc list-inside">
            <li>Profiles are imported first, then clients</li>
            <li>Existing profiles and clients will be updated</li>
            <li>New profiles and clients will be created</li>
            <li>Client references to profiles are validated before import</li>
          </ul>
        </div>

        <div className="flex gap-3">
          <button
            type="submit"
            disabled={!selectedFile}
            className="flex justify-center flex-1 px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500 disabled:bg-slate-300 disabled:cursor-not-allowed"
          >
            Import Configuration
          </button>
          <a
            href="/mqtt"
            className="flex justify-center px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md shadow-xs hover:bg-slate-50 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
          >
            Cancel
          </a>
        </div>
      </Form>
    </div>
  );
}
