import { secondaryButton } from "../ui/styles";
import { Sprite } from "../ui/Sprite";
import type { MatchAnswer, Pokemon, PossibleMatch } from "./useCollection";

/** One pending possible duplicate, shown beside what it might already be. */
export function DuplicateReview({
  match,
  remaining,
  busy,
  onAnswer,
  onStop,
}: {
  match: PossibleMatch;
  remaining: number;
  busy: boolean;
  onAnswer: (answer: MatchAnswer) => void;
  onStop: () => void;
}) {
  return (
    <div
      role="group"
      aria-label={`Possible duplicate 1 of ${remaining}`}
      className="flex flex-col gap-3 rounded-xl bg-slate-50 p-3 dark:bg-slate-950"
    >
      <p className="text-sm font-medium">
        Is this a Pokémon you already have?{" "}
        <span className="text-slate-500">{remaining} left</span>
      </p>
      <div className="grid gap-2 sm:grid-cols-2">
        <Side
          label={`In this file · line ${match.row}`}
          pokemon={match.incoming}
          compare={match.candidates[0]?.pokemon}
        />
        {match.candidates.map((c) => (
          <Side
            key={c.pokemon.id}
            label={
              c.origin === "collection"
                ? "Already in your collection"
                : `Earlier in this file · line ${c.row ?? "?"}`
            }
            pokemon={c.pokemon}
            compare={match.incoming}
          >
            <button
              type="button"
              className="w-full rounded-lg bg-teal-700 px-3 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:opacity-50"
              disabled={busy}
              onClick={() =>
                onAnswer({
                  row: match.row,
                  answer: "same",
                  sameAs: c.pokemon.id,
                })
              }
            >
              Same Pokémon
            </button>
          </Side>
        ))}
      </div>
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          className="flex-1 rounded-lg px-3 py-2 text-sm font-semibold ring-1 ring-slate-300 hover:bg-white disabled:opacity-50 dark:ring-slate-700 dark:hover:bg-slate-900"
          disabled={busy}
          onClick={() => onAnswer({ row: match.row, answer: "different" })}
        >
          Different Pokémon
        </button>
        <button type="button" className={secondaryButton} onClick={onStop}>
          Stop reviewing
        </button>
      </div>
    </div>
  );
}

function Side({
  label,
  pokemon: p,
  compare,
  children,
}: {
  label: string;
  pokemon: Pokemon;
  compare: Pokemon | undefined;
  children?: React.ReactNode;
}) {
  const facts: Array<[string, string, string | undefined]> = [
    ["CP", fmt(p.cp), compare && fmt(compare.cp)],
    ["HP", fmt(p.hp), compare && fmt(compare.hp)],
    ["Level", fmt(p.level), compare && fmt(compare.level)],
    ["IVs", ivs(p), compare && ivs(compare)],
    ["Gender", p.gender ?? "unknown", compare && (compare.gender ?? "unknown")],
    [
      "Caught",
      p.catchDate ?? "unknown",
      compare && (compare.catchDate ?? "unknown"),
    ],
  ];
  return (
    <div className="flex flex-col gap-2 rounded-lg bg-white p-3 ring-1 ring-slate-200 dark:bg-slate-900 dark:ring-slate-800">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
        {label}
      </p>
      <div className="flex items-center gap-2">
        <Sprite
          species={p.species}
          dexNumber={p.dexNumber}
          shiny={p.attributes.shiny === "yes"}
        />
        <p className="font-semibold">
          {p.species}
          {p.form !== "Normal" && (
            <span className="font-normal text-slate-500"> · {p.form}</span>
          )}
        </p>
      </div>
      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-sm">
        {facts.map(([name, value, other]) => (
          <div key={name} className="contents">
            <dt className="text-slate-500">{name}</dt>
            <dd
              className={
                other !== undefined && other !== value
                  ? "font-semibold text-amber-700 dark:text-amber-400"
                  : undefined
              }
            >
              {name === "CP" ? `CP ${value}` : value}
            </dd>
          </div>
        ))}
      </dl>
      {children}
    </div>
  );
}

const fmt = (n: number | undefined) => (n === undefined ? "?" : String(n));
const ivs = (p: Pokemon) =>
  p.iv ? `${p.iv.attack}/${p.iv.defense}/${p.iv.stamina}` : "unknown";
