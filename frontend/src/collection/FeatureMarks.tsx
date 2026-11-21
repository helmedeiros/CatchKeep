import { toast } from "sonner";
import {
  type FeatureMarks as Marks,
  type Pokemon,
  useMarkFeatures,
} from "./useCollection";

type Feature = keyof Marks;

const FEATURES: Array<[Feature, string]> = [
  ["shiny", "Shiny"],
  ["lucky", "Lucky"],
  ["shadow", "Shadow"],
  ["purified", "Purified"],
  ["costume", "Costume"],
  ["background", "Background"],
  ["dynamax", "Dynamax"],
  ["gigantamax", "Gigantamax"],
];

/**
 * The trainer's yes or no for each special feature of one Pokémon, as the
 * game shows it. Their answer outranks the export.
 */
export function FeatureMarks({
  pokemon: p,
  label,
}: {
  pokemon: Pokemon;
  label: string;
}) {
  const mark = useMarkFeatures();
  const set = (feature: Feature, value: boolean) =>
    mark.mutate(
      { id: p.id, marks: { [feature]: value } },
      { onError: () => toast.error(`Couldn't save that for ${label}.`) },
    );

  return (
    <div
      role="group"
      aria-label={`Special features of ${label}`}
      className="flex flex-col gap-2 rounded-xl bg-soft p-3 text-sm"
    >
      <p className="text-muted">
        Set what the game shows for this Pokémon. Your answer wins over the
        export, even on later imports.
      </p>
      <div className="grid grid-cols-2 gap-x-4 gap-y-2">
        {FEATURES.map(([feature, name]) => {
          const value = p.attributes[feature];
          return (
            <div
              key={feature}
              role="group"
              aria-label={name}
              className="flex flex-col gap-1"
            >
              <span className="font-medium">
                {name}
                {value === "unknown" && (
                  <span className="font-normal text-muted"> · Not set</span>
                )}
              </span>
              <div className="flex gap-1">
                {([true, false] as const).map((answer) => {
                  const pressed = value === (answer ? "yes" : "no");
                  return (
                    <button
                      key={String(answer)}
                      type="button"
                      aria-pressed={pressed}
                      disabled={mark.isPending}
                      onClick={() => set(feature, answer)}
                      className={`flex-1 rounded-full px-2 py-1 font-medium ${
                        pressed
                          ? "bg-accent text-accent-ink"
                          : "bg-surface text-muted ring-1 ring-line hover:bg-highlight"
                      }`}
                    >
                      {answer ? "Yes" : "No"}
                    </button>
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
