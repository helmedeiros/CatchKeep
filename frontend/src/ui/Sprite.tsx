import { useState } from "react";
import { spriteUrl } from "./sprites";

export function Sprite({
  species,
  dexNumber,
  shiny = false,
  size = "md",
}: {
  species: string;
  dexNumber?: number | undefined;
  shiny?: boolean;
  size?: "sm" | "md";
}) {
  const [failed, setFailed] = useState(false);
  const box = size === "sm" ? "size-10" : "size-14";
  const frame = `${box} shrink-0 rounded-xl bg-soft`;
  if (dexNumber === undefined || failed) {
    return (
      <span
        aria-hidden="true"
        className={`${frame} grid place-items-center text-lg font-semibold text-muted`}
      >
        {species.charAt(0)}
      </span>
    );
  }
  return (
    <img
      src={spriteUrl(dexNumber, shiny)}
      alt=""
      loading="lazy"
      decoding="async"
      onError={() => setFailed(true)}
      className={`${frame} object-contain p-0.5`}
    />
  );
}
