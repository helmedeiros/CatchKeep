import type { components } from "../api/schema.gen";
import { jsonResponse } from "./render";

interface Profile {
  id: string;
  displayName: string;
  pin: string;
}

type Schemas = components["schemas"];
interface Row {
  species: string;
  cp: number;
  lucky: boolean;
  /** Features confirmed absent. */
  absent?: string[];
}
interface Stored extends Row {
  id: string;
}
interface FakeImport {
  rows: Row[];
  status: "staged" | "committed";
  answers: Map<number, Schemas["MatchAnswer"]>;
  done?: { newCount: number; updatedCount: number; pending: number };
}

const toPokemon = (id: string, p: Row): Schemas["Pokemon"] => {
  const tri = (f: string) => (p.absent?.includes(f) ? "no" : "unknown");
  return {
    id,
    species: p.species,
    form: "Normal",
    cp: p.cp,
    iv: { attack: 15, defense: 14, stamina: 13 },
    attributes: {
      shiny: tri("shiny"),
      lucky: p.lucky ? "yes" : "unknown",
      shadow: "unknown",
      purified: "unknown",
      favorite: "unknown",
      costume: tri("costume"),
      background: tri("background"),
      dynamax: tri("dynamax"),
      gigantamax: tri("gigantamax"),
    },
  };
};

/**
 * An in-memory stand-in for the CatchKeep API that follows the contract
 * closely enough for component tests: profiles, PIN sign-in, session cookie,
 * imports and the collection. A staged row "might be" every collected
 * Pokémon of the same species; with one candidate it is an exact match.
 */
export function createFakeApi(options: { lockAfter?: number } = {}) {
  const profiles: Profile[] = [];
  let signedIn: Profile | null = null;
  let failures = 0;
  let next = 0;
  const collection: Stored[] = [];
  const imports = new Map<string, FakeImport>();
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
      imports.set(id, { rows, status: "staged", answers: new Map() });
      return jsonResponse(201, preview(id));
    }
    const commit = path.match(/^\/imports\/([^/]+)\/commit$/);
    if (commit && req.method === "POST") {
      const imp = imports.get(commit[1] ?? "");
      if (!imp) return err(404, "NOT_FOUND", "import not found");
      const r = review(imp);
      for (const row of r.add) {
        next += 1;
        collection.push({ id: `P${next}`, ...row });
      }
      for (const { target, row } of r.update) Object.assign(target, row);
      imp.status = "committed";
      imp.done = {
        newCount: r.add.length,
        updatedCount: r.update.length,
        pending: r.pending,
      };
      return jsonResponse(200, preview(commit[1] ?? ""));
    }
    const answer = path.match(/^\/imports\/([^/]+)\/answers$/);
    if (answer && req.method === "POST") {
      const imp = imports.get(answer[1] ?? "");
      if (!imp) return err(404, "NOT_FOUND", "import not found");
      for (const a of (body as Schemas["MatchAnswers"]).answers) {
        imp.answers.set(a.row, a);
      }
      return jsonResponse(200, preview(answer[1] ?? ""));
    }
    if (path === "/collection/confirmations" && req.method === "POST") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      const c = body as Schemas["AbsenceConfirmation"];
      let updated = 0;
      for (const p of collection) {
        if (p.species !== c.species) continue;
        const add = c.features.filter((f) => !p.absent?.includes(f));
        if (add.length === 0) continue;
        p.absent = [...(p.absent ?? []), ...add];
        updated += 1;
      }
      return jsonResponse(200, { updated });
    }
    if (path === "/collection") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      const sorted = [...collection].sort((a, b) => b.cp - a.cp);
      return jsonResponse(200, {
        total: sorted.length,
        pokemon: sorted.map((p) => toPokemon(p.id, p)),
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

  function review(imp: FakeImport) {
    const add: Row[] = [];
    const update: Array<{ target: Stored; row: Row }> = [];
    const matches: Schemas["PossibleMatch"][] = [];
    imp.rows.forEach((row, i) => {
      const line = i + 2;
      const candidates = collection.filter((p) => p.species === row.species);
      if (candidates.length === 0) {
        add.push(row);
        return;
      }
      const a = imp.answers.get(line);
      const target = candidates.find((c) => c.id === a?.sameAs);
      if (a?.answer === "different") add.push(row);
      if (a?.answer === "same" && target) update.push({ target, row });
      matches.push({
        row: line,
        incoming: toPokemon(`row-${line}`, row),
        candidates: candidates.map((c) => ({
          origin: "collection",
          pokemon: toPokemon(c.id, c),
        })),
        exact: candidates.length === 1,
        answer: a?.answer ?? "pending",
        ...(a?.sameAs === undefined ? {} : { sameAs: a.sameAs }),
      });
    });
    const pending = matches.filter((m) => m.answer === "pending").length;
    return { add, update, matches, pending };
  }

  function preview(id: string): Schemas["ImportPreview"] {
    const imp = imports.get(id);
    const r = imp && !imp.done ? review(imp) : null;
    return {
      id,
      status: imp?.status ?? "staged",
      source: "poke-genie",
      totalRows: imp?.rows.length ?? 0,
      newCount: imp?.done?.newCount ?? r?.add.length ?? 0,
      updatedCount: imp?.done?.updatedCount ?? r?.update.length ?? 0,
      possibleMatchCount: imp?.done?.pending ?? r?.pending ?? 0,
      possibleMatches: r?.matches ?? [],
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
