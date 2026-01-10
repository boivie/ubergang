import {
  IconLoader2,
  IconCircleCheck,
  IconExclamationCircle,
} from "@tabler/icons-react";
import { useState } from "react";
import { Form } from "react-router";
import { useApiService } from "../api/api_client";

type State =
  | { type: "input" }
  | { type: "loading" }
  | { type: "success"; adminFqdn: string }
  | { type: "error"; message: string };

export default function SetupComponent() {
  const api = useApiService();
  const [email, setEmail] = useState("");
  const [domain, setDomain] = useState("");
  const [state, setState] = useState<State>({ type: "input" });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Basic validation
    if (!email || !domain) {
      setState({
        type: "error",
        message: "Please fill in all required fields.",
      });
      return;
    }

    // Email validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      setState({
        type: "error",
        message: "Please enter a valid email address.",
      });
      return;
    }

    // Domain validation (basic)
    const domainRegex = /^[a-zA-Z0-9][a-zA-Z0-9-_.]*[a-zA-Z0-9]\.[a-zA-Z]{2,}$/;
    if (!domainRegex.test(domain)) {
      setState({
        type: "error",
        message: "Please enter a valid domain name (e.g., example.com).",
      });
      return;
    }

    setState({ type: "loading" });

    try {
      const response = await api.BootstrapConfigure({
        email,
        siteFqdn: domain,
      });

      if (response.admin_fqdn) {
        setState({
          type: "success",
          adminFqdn: response.admin_fqdn,
        });
      } else {
        setState({
          type: "error",
          message: "Configuration failed. Please try again.",
        });
      }
    } catch (error) {
      setState({
        type: "error",
        message:
          "Failed to configure server. Please check your connection and try again.",
      });
    }
  };

  const renderContent = () => {
    switch (state.type) {
      case "input":
        return (
          <div className="space-y-6">
            <div className="text-center">
              <p className="text-slate-600 text-sm leading-relaxed">
                Welcome to Ubergang! This is your first time running the server,
                so let's get you set up.
              </p>
            </div>
            <Form className="space-y-6" onSubmit={handleSubmit}>
              <div>
                <label
                  htmlFor="email"
                  className="block text-sm font-medium text-slate-700"
                >
                  Administrator Email
                </label>
                <div className="mt-1">
                  <input
                    id="email"
                    type="email"
                    placeholder="admin@example.com"
                    className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
                    name="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                  />
                </div>
                <p className="mt-1 text-xs text-slate-500">
                  Used for TLS certificate registration and first user.
                </p>
              </div>
              <div>
                <label
                  htmlFor="domain"
                  className="block text-sm font-medium text-slate-700"
                >
                  Site Domain (FQDN)
                </label>
                <div className="mt-1">
                  <input
                    id="domain"
                    type="text"
                    placeholder="example.com"
                    className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
                    name="domain"
                    value={domain}
                    onChange={(e) => setDomain(e.target.value)}
                    required
                  />
                </div>
                <p className="mt-1 text-xs text-slate-500">
                  Your root domain (admin will be at account.
                  {domain || "yourdomain.com"})
                </p>
              </div>
              <button
                type="submit"
                className="w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500"
              >
                Complete Setup
              </button>
            </Form>
          </div>
        );

      case "loading":
        return (
          <div className="flex flex-col items-center justify-center space-y-4 py-8">
            <IconLoader2 className="animate-spin text-emerald-500" size={48} />
            <p className="text-lg text-slate-700">Configuring server...</p>
            <p className="text-sm text-slate-500">
              This will only take a moment
            </p>
          </div>
        );

      case "success":
        return (
          <div className="space-y-6">
            <div className="flex flex-col items-center space-y-4 py-4">
              <IconCircleCheck className="text-emerald-500" size={64} />
              <h2 className="text-xl font-semibold text-slate-800">
                Setup Complete!
              </h2>
              <div className="text-center space-y-2 max-w-md">
                <p className="text-slate-600">The changes have been saved.</p>
                <div className="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-md">
                  <p className="text-sm text-blue-800 font-medium">
                    Important: Please restart the server now
                  </p>
                  <p className="text-xs text-blue-600 mt-2">
                    The configuration will take effect after restart
                  </p>
                </div>
              </div>
            </div>
          </div>
        );

      case "error":
        return (
          <div className="space-y-6">
            <div className="flex flex-col items-center space-y-4 py-4">
              <IconExclamationCircle className="text-red-500" size={48} />
              <p className="text-lg text-red-700 text-center">
                {state.message}
              </p>
            </div>
            <button
              type="button"
              onClick={() => setState({ type: "input" })}
              className="w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-red-600 hover:bg-red-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
            >
              Try Again
            </button>
          </div>
        );
    }
  };

  return (
    <section className="bg-gray-50 min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="w-full max-w-md bg-white rounded-lg shadow-lg md:mt-0 xl:p-0">
        <div className="p-6 space-y-6 sm:p-8">
          <div className="text-center">
            <h1 className="text-2xl leading-tight tracking-tight text-slate-800 md:text-3xl">
              <span className="flex items-center gap-1.5 whitespace-nowrap py-3 focus:outline-hidden lg:flex-1">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  className="icon icon-tabler icon-tabler-shield-code"
                  width="48"
                  height="48"
                  viewBox="0 0 24 24"
                  strokeWidth="1.5"
                  stroke="#10b981"
                  fill="none"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path stroke="none" d="M0 0h24v24H0z" fill="none" />
                  <path d="M12 21a12 12 0 0 1 -8.5 -15a12 12 0 0 0 8.5 -3a12 12 0 0 0 8.5 3a12 12 0 0 1 -.078 7.024" />
                  <path d="M20 21l2 -2l-2 -2" />
                  <path d="M17 17l-2 2l2 2" />
                </svg>
                Übergang
              </span>
            </h1>
          </div>
          {renderContent()}
        </div>
      </div>
    </section>
  );
}
