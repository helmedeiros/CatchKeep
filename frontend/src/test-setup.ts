import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { toast } from "sonner";
import { afterEach, beforeEach } from "vitest";
import { resetAllFilters } from "./collection/useFilters";
import { resetTicks } from "./plan/ticks";

// jsdom lacks pointer capture, which Sonner's toasts use on press.
Element.prototype.setPointerCapture ??= () => {};
Element.prototype.releasePointerCapture ??= () => {};
Element.prototype.hasPointerCapture ??= () => false;

afterEach(() => {
  toast.dismiss();
  cleanup();
});

beforeEach(() => {
  window.history.replaceState({}, "", "/");
  window.localStorage.clear();
  resetAllFilters();
  resetTicks();
});
