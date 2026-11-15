import type { ReactNode } from "react";
import type { Api } from "./client";
import { ApiContext } from "./context";

export function ApiProvider({
  api,
  children,
}: {
  api: Api;
  children: ReactNode;
}) {
  return <ApiContext value={api}>{children}</ApiContext>;
}
