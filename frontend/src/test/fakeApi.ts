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
  protected?: boolean;
  /** Features confirmed absent. */
  absent?: string[];
  /** Features the trainer marked yes or no for this Pokémon. */
  marks?: Record<string, boolean>;
}
interface Stored extends Row {
  id: string;
}
interface FakeImport {
  rows: Row[];
  /** A file that is not a Poke Genie export, read once mapped. */
  table?: { header: string[]; records: string[][] };
  mapping?: Schemas["ColumnMapEntry"][];
  status: "staged" | "committed";
  answers: Map<number, Schemas["MatchAnswer"]>;
  done?: { newCount: number; updatedCount: number; pending: number };
}

const toPokemon = (id: string, p: Row): Schemas["Pokemon"] => {
  const tri = (f: string, otherwise: "yes" | "unknown" = "unknown") => {
    const mark = p.marks?.[f];
    if (mark !== undefined) return mark ? "yes" : "no";
    return p.absent?.includes(f) ? "no" : otherwise;
  };
  return {
    id,
    species: p.species,
    form: "Normal",
    dexNumber: p.dexNumber,
    protected: p.protected ?? false,
    cp: p.cp,
    hp: p.hp,
    level: 20,
    iv: { attack: 15, defense: 14, stamina: 13 },
    attributes: {
      shiny: tri("shiny"),
      lucky: tri("lucky", p.lucky ? "yes" : "unknown"),
      shadow: tri("shadow"),
      purified: tri("purified"),
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
/** admin: whether the app runs on the host Mac (default true). */
export function createFakeApi(
  options: { lockAfter?: number; admin?: boolean } = {},
) {
  const admin = options.admin ?? true;
  const profiles: Profile[] = [];
  let signedIn: Profile | null = null;
  let failures = 0;
  let next = 0;
  let setup: Schemas["Setup"] | null = null;
  let speciesRules: Schemas["SpeciesRule"][] = [];
  let categoryRules: Schemas["CategoryRule"][] = [];
  let teams: Schemas["Team"][] = [];
  const collection: Stored[] = [];
  const imports = new Map<string, FakeImport>();
  const mappings = new Map<string, Schemas["ColumnMapEntry"][]>();
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
    if (path === "/host") {
      return jsonResponse(200, {
        admin,
        addresses: admin ? ["http://192.168.1.20:8080"] : [],
      });
    }
    const pinPath = path.match(/^\/profiles\/([^/]+)\/pin$/);
    if (pinPath && req.method === "PUT") {
      if (!admin)
        return err(
          403,
          "ADMIN_ONLY",
          "Only the computer running CatchKeep can do this.",
        );
      const p = profiles.find((x) => x.id === pinPath[1]);
      if (!p) return err(404, "NOT_FOUND", "No such profile.");
      p.pin = (body as Schemas["NewPin"]).pin;
      if (signedIn?.id === p.id) signedIn = null;
      failures = 0;
      return new Response(null, { status: 204 });
    }
    if (path === "/profiles" && req.method === "POST") {
      if (!admin)
        return err(
          403,
          "ADMIN_ONLY",
          "Only the computer running CatchKeep can do this.",
        );
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
        const columns = header?.split(",") ?? [];
        if (columns.length < 2 || lines.length === 0) {
          return err(
            422,
            "UNSUPPORTED_FILE",
            "this file is not a CSV export CatchKeep can read: it needs a header row with at least two columns and one Pokémon per line",
          );
        }
        const id = `01JD2Z8Q4M8V7Y6X5W4T3S2T${String(imports.size).padStart(2, "0")}`;
        const imp: FakeImport = {
          rows: [],
          status: "staged",
          answers: new Map(),
          table: { header: columns, records: lines.map((l) => l.split(",")) },
        };
        const saved = mappings.get(signature(columns));
        if (saved) mapTable(imp, saved);
        imports.set(id, imp);
        return jsonResponse(201, preview(id));
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
    const mapPath = path.match(/^\/imports\/([^/]+)\/mapping$/);
    if (mapPath && req.method === "PUT") {
      const imp = imports.get(mapPath[1] ?? "");
      if (!imp?.table) return err(404, "NOT_FOUND", "import not found");
      const entries = (body as Schemas["ColumnMappingInput"]).mapping;
      if (!entries.some((e) => e.field === "species")) {
        return err(422, "VALIDATION_FAILED", "choose the species column");
      }
      mapTable(imp, entries);
      mappings.set(signature(imp.table.header), entries);
      return jsonResponse(200, preview(mapPath[1] ?? ""));
    }
    const commit = path.match(/^\/imports\/([^/]+)\/commit$/);
    if (commit && req.method === "POST") {
      const imp = imports.get(commit[1] ?? "");
      if (!imp) return err(404, "NOT_FOUND", "import not found");
      if (imp.table && !imp.mapping) {
        return err(
          409,
          "IMPORT_NEEDS_MAPPING",
          "Map the file's columns first.",
        );
      }
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
    if (path === "/capacity-bands") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      return jsonResponse(200, {
        bands: [...BANDS],
      });
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
    const features = /^\/collection\/([^/]+)\/features$/.exec(path);
    if (features && req.method === "PUT") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      const p = collection.find((x) => x.id === features[1]);
      if (!p) return err(404, "NOT_FOUND", "not in your collection");
      const marks = { ...(body as Record<string, boolean>) };
      if (marks.shadow && marks.purified) {
        return err(
          422,
          "VALIDATION_FAILED",
          "cannot be both shadow and purified",
        );
      }
      if (marks.shadow) marks.purified = false;
      if (marks.purified) marks.shadow = false;
      p.marks = { ...p.marks, ...marks };
      return jsonResponse(200, toPokemon(p.id, p));
    }
    const protect = /^\/collection\/([^/]+)\/protection$/.exec(path);
    if (protect && req.method === "PUT") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      const p = collection.find((x) => x.id === protect[1]);
      if (!p) return err(404, "NOT_FOUND", "not in your collection");
      p.protected = (body as Schemas["Protection"]).protected;
      return jsonResponse(200, { protected: p.protected });
    }
    if (path === "/teams" || path.startsWith("/teams/")) {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      const id = path.split("/")[2];
      const input = body as Schemas["TeamInput"];
      const taken = (other?: string) =>
        teams.some(
          (t) =>
            t.id !== other &&
            t.name.toLowerCase() === input.name.trim().toLowerCase(),
        );
      if (req.method === "GET") return jsonResponse(200, { teams });
      if (req.method === "POST") {
        if (taken()) return err(409, "TEAM_NAME_TAKEN", "name is used");
        const team = {
          id: `T${++next}`,
          name: input.name.trim(),
          memberIds: input.memberIds,
        };
        teams = [...teams, team];
        return jsonResponse(201, team);
      }
      if (!teams.some((t) => t.id === id)) {
        return err(404, "NOT_FOUND", "team not found");
      }
      if (req.method === "DELETE") {
        teams = teams.filter((t) => t.id !== id);
        return new Response(null, { status: 204 });
      }
      if (taken(id)) return err(409, "TEAM_NAME_TAKEN", "name is used");
      const team = {
        id: id ?? "",
        name: input.name.trim(),
        memberIds: input.memberIds,
      };
      teams = teams.map((t) => (t.id === id ? team : t));
      return jsonResponse(200, team);
    }
    if (path === "/species-rules") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      if (!setup) return err(409, "SETUP_REQUIRED", "Set up first.");
      if (req.method === "PUT") {
        speciesRules = (body as Schemas["SpeciesRules"]).rules;
      }
      return jsonResponse(200, { rules: speciesRules });
    }
    if (path === "/category-rules") {
      if (!signedIn) return err(401, "UNAUTHENTICATED", "Not signed in.");
      if (!setup) return err(409, "SETUP_REQUIRED", "Set up first.");
      if (req.method === "PUT") {
        categoryRules = (body as Schemas["CategoryRules"]).rules;
      }
      return jsonResponse(200, { rules: categoryRules });
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
   * Mewtwo (legendary) and Mew (mythical) are rare here.
   */
  const BANDS = [
    { upTo: 500, ordinary: 1, rare: 2 },
    { upTo: 1500, ordinary: 1, rare: 3 },
    { upTo: 3000, ordinary: 2, rare: 4 },
    { upTo: 6000, ordinary: 2, rare: 5 },
    { upTo: 20000, ordinary: 3, rare: 6 },
  ] as const;

  const CATEGORIES: Record<string, Schemas["SpeciesCategory"]> = {
    Mewtwo: "legendary",
    Mew: "mythical",
  };

  function keepSummary(s: Schemas["Setup"]): Schemas["KeepSummary"] {
    const band = BANDS.find((b) => s.capacity <= b.upTo) ?? BANDS[0];
    const ordinary = band.ordinary;
    const rareMinimum = band.rare;
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
      const onTeam = new Set(teams.flatMap((t) => t.memberIds));
      const isProtected = (p: Stored) =>
        (p.protected ?? false) || onTeam.has(p.id);
      const best = [...members].sort(
        (a, b) =>
          Number(isProtected(b)) - Number(isProtected(a)) || b.cp - a.cp,
      );
      const protectedIds = best.filter(isProtected).map((p) => p.id);
      const category = CATEGORIES[species];
      const rare = category !== undefined;
      const own =
        speciesRules.find((r) => r.species === species) ??
        categoryRules.find((r) => r.category === category);
      const ownRule = own && "species" in own ? "species" : "category";
      const tradeBonus = rare && s.goals.includes("trading");
      const generated = (rare ? rareMinimum : ordinary) + Number(tradeBonus);
      const minimum = own?.favorite
        ? members.length
        : own
          ? own.minimum
          : generated;
      const rule: Schemas["VariantGroup"]["rule"] = own
        ? own.favorite && ownRule === "species"
          ? "favorite"
          : ownRule
        : rare
          ? "rare"
          : "default";
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
        ...(category && { category }),
        default: generated,
        tradeBonus,
        rule,
        count: members.length,
        minimum,
        keep: Math.min(Math.max(minimum, protectedIds.length), members.length),
        keptIds: best
          .slice(0, Math.max(minimum, protectedIds.length))
          .map((p) => p.id),
        surplusIds: best
          .slice(Math.max(minimum, protectedIds.length))
          .map((p) => p.id),
        protectedIds,
      };
    });
    out.sort((a, b) => b.surplusIds.length - a.surplusIds.length);
    const surplus = out.reduce((n, g) => n + g.surplusIds.length, 0);
    return {
      total: collection.length,
      kept: collection.length - surplus - needsConfirmation,
      surplus,
      needsConfirmation,
      protected: out.reduce((n, g) => n + g.protectedIds.length, 0),
      minimum: ordinary,
      rareMinimum,
      groups: out,
    };
  }

  function signature(header: string[]) {
    return header.map((h) => h.trim().toLowerCase()).join("\n");
  }

  /** Reads species, Pokédex number, CP and HP columns; ignores the rest. */
  function mapTable(imp: FakeImport, entries: Schemas["ColumnMapEntry"][]) {
    const table = imp.table;
    if (!table) return;
    const at = (field: string) =>
      table.header.indexOf(
        entries.find((e) => e.field === field)?.column ?? "",
      );
    imp.mapping = entries;
    imp.rows = table.records.map((r) => ({
      species: r[at("species")] ?? "",
      dexNumber: Number(r[at("dex_number")] ?? 0),
      cp: Number(r[at("cp")] ?? 0),
      hp: Number(r[at("hp")] ?? 0),
      lucky: false,
    }));
  }

  /** Suggests fields from a few common column names. */
  function suggest(header: string[]): Schemas["ColumnMapEntry"][] {
    const names: Record<string, Schemas["MappableField"]> = {
      name: "species",
      species: "species",
      cp: "cp",
      "combat power": "cp",
      hp: "hp",
    };
    return header.flatMap((column) => {
      const field = names[column.trim().toLowerCase()];
      return field ? [{ column, field }] : [];
    });
  }

  function preview(id: string): Schemas["ImportPreview"] {
    const imp = imports.get(id);
    const r =
      imp && !imp.done && !(imp.table && !imp.mapping) ? review(imp) : null;
    const table = imp?.table;
    return {
      id,
      status: imp?.status ?? "staged",
      source: table ? "csv" : "poke-genie",
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
      needsMapping: table !== undefined && imp?.mapping === undefined,
      columns: (table?.header ?? []).map((name, i) => ({
        name,
        samples:
          table?.records
            .map((r) => r[i] ?? "")
            .filter((v) => v !== "")
            .slice(0, 3) ?? [],
      })),
      mapping: table ? (imp?.mapping ?? suggest(table.header)) : [],
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
