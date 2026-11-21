import { cva } from "class-variance-authority";

/** Buttons in the chosen direction: pill-shaped, accent for the main one. */
export const button = cva(
  "inline-flex items-center justify-center gap-2 rounded-full font-semibold outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-ground disabled:opacity-50",
  {
    variants: {
      intent: {
        primary: "bg-accent px-5 py-2.5 text-accent-ink hover:bg-accent/90",
        secondary: "px-3 py-2 text-sm text-accent hover:bg-highlight",
        outline:
          "px-3 py-1.5 text-sm text-accent ring-1 ring-line hover:bg-highlight",
      },
      /** A primary button fills its row unless it fits its content. */
      fit: { true: "", false: "" },
    },
    compoundVariants: [{ intent: "primary", fit: false, class: "w-full" }],
    defaultVariants: { intent: "primary", fit: false },
  },
);

export const card =
  "rounded-[20px] bg-surface p-4 shadow-card ring-1 ring-line";
export const input =
  "w-full rounded-full border border-line bg-surface px-4 py-2 text-base outline-none placeholder:text-muted focus:border-accent focus:ring-2 focus:ring-accent/30";
export const primaryButton = button({ intent: "primary" });
export const secondaryButton = button({ intent: "secondary" });
export const outlineButton = button({ intent: "outline" });
export const alert = "rounded-xl bg-warn/15 px-3 py-2 text-sm text-warn";
