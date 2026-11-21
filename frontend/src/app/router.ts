import { useSyncExternalStore } from "react";

const NAVIGATE = "catchkeep:navigate";

function subscribe(onChange: () => void) {
  window.addEventListener("popstate", onChange);
  window.addEventListener(NAVIGATE, onChange);
  return () => {
    window.removeEventListener("popstate", onChange);
    window.removeEventListener(NAVIGATE, onChange);
  };
}

/** The current place, from the address bar; back and forward follow it. */
export function usePath(): string {
  return useSyncExternalStore(subscribe, () => window.location.pathname);
}

/** Goes to another place without reloading, keeping it in the history. */
export function navigate(to: string) {
  if (window.location.pathname === to) return;
  window.history.pushState({}, "", to);
  window.dispatchEvent(new Event(NAVIGATE));
  window.scrollTo({ top: 0 });
}
