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
  dexNumber: number;
  cp: number;
  hp: number;
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
    dexNumber: p.dexNumber,
    cp: p.cp,
    hp: p.hp,
    level: 20,
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
  let setup: Schemas["Setup"] | null = null;
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
        return {
          species: c[1] ?? "",
          dexNumber: Number(c[3]),
          cp: Number(c[5]),
          hp: Number(c[6]),
          lucky: c[21] === "1",
        };
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
    if (path === "/setup") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      if (req.method === "PUT") {
        setup = body as Schemas["Setup"];
        return jsonResponse(200, setup);
      }
      return setup
        ? jsonResponse(200, setup)
        : err(404, "NOT_FOUND", "setup not done yet");
    }
    if (path === "/keep-summary") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      if (!setup) return err(409, "SETUP_REQUIRED", "Set up first.");
      return jsonResponse(200, keepSummary(setup));
    }
    if (path === "/collection/confirmations" && req.method === "POST") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      const c = body as Schemas["AbsenceConfirmation"];
      let updated = 0;
      for (const p of collection) {
        if (c.species !== undefined && p.species !== c.species) continue;
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

  /**
   * Groups by species once all five variant features are confirmed. Only
   * Mewtwo is rare here.
   */
  function keepSummary(s: Schemas["Setup"]): Schemas["KeepSummary"] {
    const band = s.capacity <= 1000 ? 1 : s.capacity <= 4000 ? 2 : 3;
    const ordinary = band;
    const rareMinimum = band * 2;
    const groups = new Map<string, Stored[]>();
    let needsConfirmation = 0;
    for (const p of collection) {
      if ((p.absent?.length ?? 0) < 5) {
        needsConfirmation += 1;
        continue;
      }
      groups.set(p.species, [...(groups.get(p.species) ?? []), p]);
    }
    const out = [...groups.entries()].map(([species, members]) => {
      const best = [...members].sort((a, b) => b.cp - a.cp);
      const rare = species === "Mewtwo";
      const minimum = rare ? rareMinimum : ordinary;
      return {
        species,
        form: "Normal",
        shiny: false,
        costume: false,
        background: false,
        dynamax: false,
        gigantamax: false,
        lucky: false,
        shadow: false,
        purified: false,
        rare,
        count: members.length,
        minimum,
        keep: Math.min(minimum, members.length),
        keptIds: best.slice(0, minimum).map((p) => p.id),
        surplusIds: best.slice(minimum).map((p) => p.id),
      };
    });
    out.sort((a, b) => b.surplusIds.length - a.surplusIds.length);
    const surplus = out.reduce((n, g) => n + g.surplusIds.length, 0);
    return {
      total: collection.length,
      kept: collection.length - surplus - needsConfirmation,
      surplus,
      needsConfirmation,
      minimum: ordinary,
      rareMinimum,
      groups: out,
    };
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
