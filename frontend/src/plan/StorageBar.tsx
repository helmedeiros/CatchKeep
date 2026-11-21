import type { Plan } from "./usePlan";

/** How full storage is against the trainer's limit, as a bar and a line. */
export function StorageBar({ use }: { use: Plan["use"] }) {
  const pct =
    use.capacity > 0 ? Math.round((use.occupied / use.capacity) * 100) : 0;
  const limitPct =
    use.capacity > 0 ? Math.round((use.limit / use.capacity) * 100) : 0;
  const over = use.occupied > use.limit;
  return (
    <div className="flex flex-col gap-1.5 text-sm">
      <div
        className="h-2.5 overflow-hidden rounded-full bg-soft"
        aria-hidden="true"
      >
        <div
          className={`h-full ${over ? "bg-warn" : "bg-accent"}`}
          style={{ width: `${Math.min(100, pct)}%` }}
        />
      </div>
      <p className="text-muted">
        {use.occupied} of {use.capacity} slots used ({pct}%); your limit is{" "}
        {limitPct}% ({use.limit} slots).
        {use.estimated &&
          " This is an estimate from your import; add the number the game shows in your setup for an exact one."}
      </p>
    </div>
  );
}
