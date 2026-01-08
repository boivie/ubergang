import { createContext } from "react";
import { WebauthnService } from "./webauthn-service";

export const WebauthnServiceContext = createContext<
  WebauthnService | undefined
>(undefined);
