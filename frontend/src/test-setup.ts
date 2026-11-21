import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach } from "vitest";
import { resetAllFilters } from "./collection/useFilters";

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  window.history.replaceState({}, "", "/");
  resetAllFilters();
});
