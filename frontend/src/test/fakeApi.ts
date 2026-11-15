import type { components } from "../api/schema.gen";
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
  const collection: Array<{
    id: string;
    species: string;
    cp: number;
    lucky: boolean;
  }> = [];
  const imports = new Map<
    string,
    {
      rows: Array<{ species: string; cp: number; lucky: boolean }>;
      status: "staged" | "committed";
    }
  >();
  const err = (status: number, code: string, message: string) =>
    jsonResponse(status, { error: { code, message, correlationId: "test" } });

  const fetchImpl: typeof fetch = async (input) => {
    const req = new Request(input);
    const path = new URL(req.url).pathname.replace(/^\/api\/v1/, "");
    const isCsv =
      req.headers.get("Content-Type")?.startsWith("text/csv") ?? false;
    const text =
      req.method === "GET" || req.method === "DELETE" ? "" : await req.text();
    const body = isCsv || text === "" ? null : JSON.parse(text);

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
    if (path === "/imports" && req.method === "POST") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      const [header, ...lines] = text.trim().split("\n");
      if (!header?.includes("Pokemon Number")) {
        return err(
          422,
          "UNSUPPORTED_FILE",
          "this file is not a Poke Genie CSV export",
        );
      }
      const rows = lines.map((l) => {
        const c = l.split(",");
        return { species: c[1] ?? "", cp: Number(c[5]), lucky: c[21] === "1" };
      });
      const id =
        `01JD2Z8Q4M8V7Y6X5W4T3S2I${String(imports.size).padStart(2, "0")}`.replace(
          "I",
          "R",
        );
      imports.set(id, { rows, status: "staged" });
      return jsonResponse(201, preview(id));
    }
    const commit = path.match(/^\/imports\/([^/]+)\/commit$/);
    if (commit && req.method === "POST") {
      const imp = imports.get(commit[1] ?? "");
      if (!imp) return err(404, "NOT_FOUND", "import not found");
      imp.status = "committed";
      for (const r of imp.rows) {
        next += 1;
        collection.push({ id: `P${next}`, ...r });
      }
      return jsonResponse(200, preview(commit[1] ?? ""));
    }
    if (path === "/collection") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      const sorted = [...collection].sort((a, b) => b.cp - a.cp);
      return jsonResponse(200, {
        total: sorted.length,
        pokemon: sorted.map((p) => ({
          id: p.id,
          species: p.species,
          form: "Normal",
          cp: p.cp,
          iv: { attack: 15, defense: 14, stamina: 13 },
          attributes: {
            shiny: "unknown",
            lucky: p.lucky ? "yes" : "unknown",
            shadow: "unknown",
            purified: "unknown",
            favorite: "unknown",
            costume: "unknown",
            background: "unknown",
            dynamax: "unknown",
            gigantamax: "unknown",
          },
        })),
      });
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

  function preview(id: string): components["schemas"]["ImportPreview"] {
    const imp = imports.get(id);
    return {
      id,
      status: imp?.status ?? "staged",
      source: "poke-genie",
      totalRows: imp?.rows.length ?? 0,
      newCount: imp?.rows.length ?? 0,
      updatedCount: 0,
      possibleMatchCount: 0,
      possibleMatches: [],
      invalidRows: [],
      ignoredColumns: ["Dust"],
      unknownFields: [
        "shiny",
        "costume",
        "background",
        "dynamax",
        "gigantamax",
      ],
    };
  }

  return {
    fetch: fetchImpl,
    signIn(displayName: string, pin = "2468") {
      next += 1;
      const p = {
        id: `01JD2Z8Q4M8V7Y6X5W4T3S2R${String(next).padStart(2, "0")}`,
        displayName,
        pin,
      };
      profiles.push(p);
      signedIn = p;
    },
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
