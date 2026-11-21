/**
 * Keepy, CatchKeep's own mascot: a friendly storage box. Original art in
 * the theme's colours; never based on Pokémon artwork.
 */
export function Keepy({ size = 104 }: { size?: number }) {
  return (
    <svg
      viewBox="0 0 120 110"
      width={size}
      height={(size * 110) / 120}
      aria-hidden="true"
      className="shrink-0"
    >
      <g
        stroke="var(--ck-ink)"
        strokeWidth="3.5"
        strokeLinejoin="round"
        strokeLinecap="round"
      >
        <path d="M22 46 L98 46 L92 100 L28 100 Z" fill="var(--ck-keepy)" />
        <path
          d="M14 34 L106 34 L102 50 L18 50 Z"
          fill="var(--ck-accent)"
          transform="rotate(-6 60 42)"
        />
        <circle cx="46" cy="70" r="6.5" fill="var(--ck-ink)" stroke="none" />
        <circle cx="74" cy="70" r="6.5" fill="var(--ck-ink)" stroke="none" />
        <circle cx="48" cy="68" r="2" fill="#fff" stroke="none" />
        <circle cx="76" cy="68" r="2" fill="#fff" stroke="none" />
        <path d="M52 84 Q60 91 68 84" fill="none" />
        <ellipse cx="36" cy="82" rx="5" ry="3" fill="#f6b3a2" stroke="none" />
        <ellipse cx="84" cy="82" rx="5" ry="3" fill="#f6b3a2" stroke="none" />
        <path d="M98 62 Q112 56 110 44" fill="none" />
        <path d="M22 62 Q10 60 8 70" fill="none" />
        <path
          d="M104 22 l3 -8 l3 8 l8 3 l-8 3 l-3 8 l-3 -8 l-8 -3 Z"
          fill="var(--ck-accent)"
        />
      </g>
    </svg>
  );
}
