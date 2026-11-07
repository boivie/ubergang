import { IconDeviceTv } from "@tabler/icons-react";
import { useState } from "react";
import { Form, useNavigate } from "react-router";
import OTPInput from "../components/otp_input";

// Where a signed-in user enters the code shown on a kiosk's sign-in page.
export default function Kiosk() {
  const navigate = useNavigate();
  const [code, setCode] = useState("");

  return (
    <section className="bg-gray-50 min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="w-full max-w-md bg-white rounded-lg shadow-lg md:mt-0 xl:p-0">
        <div className="p-6 space-y-6 sm:p-8">
          <div className="text-center">
            <h1 className="text-2xl font-bold leading-tight tracking-tight text-slate-800 md:text-3xl">
              Sign In a Device
            </h1>
            <p className="mt-2 text-sm text-slate-600">Secure Proxy</p>
          </div>
          <div className="text-center">
            <IconDeviceTv className="mx-auto text-emerald-500 mb-4" size={48} />
            <p className="text-slate-600">
              Enter the 6-digit code shown on the device you want to sign in.
            </p>
          </div>
          <Form className="space-y-6" method="POST">
            <div>
              <label htmlFor="code" className="sr-only">
                Sign-in code
              </label>
              <OTPInput
                numInputs={6}
                onChange={(value) => setCode(value)}
                value={code}
                shouldAutoFocus
              />
            </div>
            <button
              type="submit"
              onClick={(e) => {
                e.preventDefault();
                navigate("/kiosk/" + code);
              }}
              disabled={code.length !== 6}
              className="w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Continue
            </button>
          </Form>
        </div>
      </div>
    </section>
  );
}
