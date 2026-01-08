import { useContext } from "react";
import { WebauthnService } from "./webauthn-service";
import { WebauthnServiceContext } from "./webauthn";

export const useWebauthnService = () => {
  const context = useContext<WebauthnService | undefined>(
    WebauthnServiceContext,
  );
  if (context === undefined) {
    throw new Error("WebauthnServiceContext must wrap this component");
  }
  return context;
};
