import { encode } from "uqr";

/**
 * A QR code drawn as SVG in the page's own colours, made in the browser
 * with no network. A phone camera reading it opens the address.
 */
export function QrCode({ text, size = 168 }: { text: string; size?: number }) {
  const { data } = encode(text, { border: 2 });
  const cells = data.length;
  let path = "";
  data.forEach((row, y) =>
    row.forEach((on, x) => {
      if (on) path += `M${x} ${y}h1v1h-1z`;
    }),
  );
  return (
    <svg
      role="img"
      aria-label={`QR code for ${text}`}
      viewBox={`0 0 ${cells} ${cells}`}
      width={size}
      height={size}
      shapeRendering="crispEdges"
      className="rounded-xl bg-white text-[#0a0d14]"
    >
      <path d={path} fill="currentColor" />
    </svg>
  );
}
