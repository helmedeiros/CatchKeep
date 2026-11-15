import { useContext } from "react";
import type { Api } from "./client";
import { ApiContext } from "./context";

export function useApi(): Api {
  const api = useContext(ApiContext);
  if (!api) {
    throw new Error("useApi must be used inside <ApiProvider>");
  }
  return api;
}
