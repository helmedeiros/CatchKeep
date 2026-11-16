/** IV percentage as a ring, green from 80%, like the scanners trainers know. */
export function IvRing({ percent }: { percent: number | null }) {
  const r = 15.9155; // circumference 100
  const tone =
    percent === null
      ? "text-slate-300 dark:text-slate-700"
      : percent >= 80
        ? "text-green-600"
        : "text-sky-600";
  return (
    <span
      className={`relative grid size-12 shrink-0 place-items-center ${tone}`}
      aria-label={percent === null ? "IVs unknown" : `${percent}% IVs`}
    >
      <svg viewBox="0 0 36 36" className="absolute inset-0 -rotate-90">
        <circle
          cx="18"
          cy="18"
          r={r}
          fill="none"
          strokeWidth="3.5"
          className="stroke-slate-200 dark:stroke-slate-800"
        />
        {percent !== null && (
          <circle
            cx="18"
            cy="18"
            r={r}
            fill="none"
            strokeWidth="3.5"
            stroke="currentColor"
            strokeDasharray={`${percent} 100`}
          />
        )}
      </svg>
      <span className="text-xs font-semibold tabular-nums text-slate-800 dark:text-slate-100">
        {percent === null ? "?" : percent}
      </span>
    </span>
  );
}
