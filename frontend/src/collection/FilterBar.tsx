import { input, secondaryButton } from "../ui/styles";
import {
  type Chip,
  type Filters,
  isFiltering,
  NO_FILTERS,
  type Sort,
} from "./filters";
import { useFilters } from "./useFilters";

const CHIP_LABELS: Record<Chip, string> = {
  transfer: "Possible transfer",
  shiny: "Shiny",
  lucky: "Lucky",
  shadow: "Shadow",
  purified: "Purified",
  costume: "Costume",
  protected: "Protected",
  team: "On a team",
  unconfirmed: "Unconfirmed",
};

const SORTS: Array<[Sort, string]> = [
  ["plan", "Weakest first"],
  ["cp", "Highest CP"],
  ["iv", "Best IVs"],
  ["recent", "Newest catch"],
  ["species", "Species A to Z"],
];

/**
 * Search, filter chips, a minimum IV and sorting for one list of Pokémon.
 * Each list remembers its own choices while the trainer moves around.
 */
export function FilterBar({
  list,
  chips,
  shown,
  total,
  defaults = NO_FILTERS,
}: {
  list: string;
  chips: Chip[];
  shown: number;
  total: number;
  defaults?: Filters;
}) {
  const { filters: f, update, reset } = useFilters(list, defaults);
  const sorts = SORTS.filter(([s]) => s !== "plan" || defaults.sort === "plan");
  const toggle = (c: Chip) =>
    update({
      chips: f.chips.includes(c)
        ? f.chips.filter((x) => x !== c)
        : [...f.chips, c],
    });

  return (
    <div className="flex flex-col gap-2.5">
      <input
        type="search"
        aria-label="Search Pokémon"
        placeholder="Search name or Pokédex number"
        className={input}
        value={f.query}
        onChange={(e) => update({ query: e.target.value })}
      />
      <div
        role="group"
        aria-label="Filters"
        className="-mx-1 flex gap-2 overflow-x-auto px-1 pb-1"
      >
        {chips.map((c) => {
          const on = f.chips.includes(c);
          return (
            <button
              key={c}
              type="button"
              aria-pressed={on}
              onClick={() => toggle(c)}
              className={`shrink-0 whitespace-nowrap rounded-full px-3.5 py-1.5 text-sm font-semibold ring-1 ${
                on
                  ? "bg-accent text-accent-ink ring-accent"
                  : "bg-surface text-ink ring-line hover:bg-highlight"
              }`}
            >
              {CHIP_LABELS[c]}
            </button>
          );
        })}
      </div>
      <div className="grid grid-cols-2 gap-2 text-sm sm:flex sm:flex-wrap">
        <label className="flex flex-col gap-1">
          <span className="px-1 text-xs text-muted">IV at least</span>
          <select
            className={`${input} py-1.5 sm:w-auto`}
            value={f.minIv}
            onChange={(e) => update({ minIv: Number(e.target.value) })}
          >
            <option value={0}>Any</option>
            <option value={80}>80%</option>
            <option value={90}>90%</option>
            <option value={100}>100%</option>
          </select>
        </label>
        <label className="flex flex-col gap-1">
          <span className="px-1 text-xs text-muted">Sort by</span>
          <select
            className={`${input} py-1.5 sm:w-auto`}
            value={f.sort}
            onChange={(e) => update({ sort: e.target.value as Sort })}
          >
            {sorts.map(([s, label]) => (
              <option key={s} value={s}>
                {label}
              </option>
            ))}
          </select>
        </label>
      </div>
      <div className="flex items-center justify-between text-sm">
        <p className="text-muted" aria-live="polite">
          {isFiltering(f) ? `Showing ${shown} of ${total}` : `${total} Pokémon`}
        </p>
        {isFiltering(f) && (
          <button type="button" className={secondaryButton} onClick={reset}>
            Clear filters
          </button>
        )}
      </div>
    </div>
  );
}
