import { jsonResponse } from "./render";

interface Profile {
  id: string;
  displayName: string;
  pin: string;
}

/**
 * An in-memory stand-in for the CatchKeep API that follows the contract
 * closely enough for component tests: profiles, PIN sign-in, session cookie.
 */
export function createFakeApi(options: { lockAfter?: number } = {}) {
  const profiles: Profile[] = [];
  let signedIn: Profile | null = null;
  let failures = 0;
  let next = 0;
  const err = (status: number, code: string, message: string) =>
    jsonResponse(status, { error: { code, message, correlationId: "test" } });

  const fetchImpl: typeof fetch = async (input) => {
    const req = new Request(input);
    const path = new URL(req.url).pathname.replace(/^\/api\/v1/, "");
    const body =
      req.method === "GET" || req.method === "DELETE" ? null : await req.json();

    if (path === "/health")
      return jsonResponse(200, { status: "ok", version: "test" });
    if (path === "/profiles" && req.method === "GET") {
      return jsonResponse(
        200,
        [...profiles]
          .sort((a, b) => a.displayName.localeCompare(b.displayName))
          .map(({ id, displayName }) => ({ id, displayName })),
      );
    }
    if (path === "/profiles" && req.method === "POST") {
      if (
        profiles.some(
          (p) => p.displayName.toLowerCase() === body.displayName.toLowerCase(),
        )
      ) {
        return err(
          409,
          "DISPLAY_NAME_TAKEN",
          "display name is already used by another profile",
        );
      }
      next += 1;
      const p = {
        id: `01JD2Z8Q4M8V7Y6X5W4T3S2R${String(next).padStart(2, "0")}`,
        displayName: body.displayName,
        pin: body.pin,
      };
      profiles.push(p);
      return jsonResponse(201, { id: p.id, displayName: p.displayName });
    }
    if (path === "/session" && req.method === "POST") {
      if (options.lockAfter !== undefined && failures >= options.lockAfter) {
        return jsonResponse(429, {
          error: {
            code: "RATE_LIMITED",
            message: "Too many wrong PINs.",
            correlationId: "t",
            details: { retryAfter: "2026-11-15T12:01:00Z" },
          },
        });
      }
      const p = profiles.find((x) => x.id === body.profileId);
      if (!p || p.pin !== body.pin) {
        failures += 1;
        return err(401, "PIN_INVALID", "Wrong PIN.");
      }
      failures = 0;
      signedIn = p;
      return jsonResponse(200, { id: p.id, displayName: p.displayName });
    }
    if (path === "/session" && req.method === "DELETE") {
      signedIn = null;
      return new Response(null, { status: 204 });
    }
    if (path === "/me") {
      return signedIn
        ? jsonResponse(200, {
            id: signedIn.id,
            displayName: signedIn.displayName,
          })
        : err(401, "UNAUTHENTICATED", "Not signed in.");
    }
    return err(404, "NOT_FOUND", `no fake for ${req.method} ${path}`);
  };

  return {
    fetch: fetchImpl,
    addProfile(displayName: string, pin: string) {
      next += 1;
      profiles.push({
        id: `01JD2Z8Q4M8V7Y6X5W4T3S2R${String(next).padStart(2, "0")}`,
        displayName,
        pin,
      });
    },
  };
}
