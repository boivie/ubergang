import { useEffect, useRef, useState } from "react";
import { Form } from "react-router";

export interface EmailFormProps {
  onSubmit: (email: string) => void;
  onConditionalWebauthn: () => void;
}

export const EmailForm = ({
  onSubmit,
  onConditionalWebauthn,
}: EmailFormProps) => {
  const [email, setEmail] = useState("");

  const effectRan = useRef(false);
  useEffect(() => {
    if (effectRan.current) {
      return;
    }
    effectRan.current = true;
    onConditionalWebauthn();
  }, [onConditionalWebauthn]);

  return (
    <Form className="space-y-6" method="POST">
      <div>
        <label
          htmlFor="email"
          className="block text-sm font-medium text-slate-700 mb-2"
        >
          Email address
        </label>
        <input
          type="email"
          name="email"
          id="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoFocus={true}
          autoComplete="username webauthn"
          className="block w-full px-3 py-2 placeholder-gray-400 border border-gray-300 rounded-md shadow-xs appearance-none focus:outline-hidden focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm"
          placeholder="you@company.com"
          required={true}
          aria-describedby="email-description"
        />
      </div>
      <button
        type="submit"
        data-testid="login-button"
        onClick={(e) => {
          e.preventDefault();
          onSubmit(email);
        }}
        className="w-full px-4 py-2 text-sm font-medium text-white border border-transparent rounded-md shadow-xs bg-emerald-600 hover:bg-emerald-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed"
        disabled={!email.trim()}
        aria-describedby="signin-button-description"
      >
        Sign in
      </button>
      <p id="signin-button-description" className="sr-only">
        Sign in with your email address using passkey or secure link
      </p>
    </Form>
  );
};
