import type { ReactNode } from "react";
import type { KeepSummary, SpeciesRule } from "./useKeep";

type Group = KeepSummary["groups"][number];

const RULE_NAMES: Record<Group["rule"], string> = {
  default: "the default",
  rare: "the rare-species default",
  category: "your category rule",
  species: "your species rule",
  favorite: "your favorite",
};

const copies = (n: number) => (n === 1 ? "1 copy" : `${n} copies`);

function generated(g: Group): string {
  const base = g.default - Number(g.tradeBonus);
  const text = g.rare ? `${copies(base)} of a rare species` : copies(base);
  return g.tradeBonus ? `${text}, +1 for trading` : text;
}

function own(g: Group, rule: SpeciesRule | undefined): string {
  if (!rule) return "None";
  if (rule.favorite) return "Favorite, keep every copy";
  return `Keep at least ${rule.minimum} across every ${g.species} variant`;
}

function protection(byYou: number, onTeam: number): string {
  const parts = [
    ...(byYou > 0 ? [`${byYou} by you`] : []),
    ...(onTeam > 0 ? [`${onTeam} on a team`] : []),
  ];
  return parts.length > 0 ? parts.join(", ") : "None";
}

function used(g: Group): string {
  const text = `${g.minimum}, from ${RULE_NAMES[g.rule]}`;
  if (g.keep > g.minimum) return `${text}; protection keeps ${g.keep}`;
  if (g.count < g.minimum)
    return `${text}; only ${copies(g.count)}, so all stay`;
  return text;
}

/**
 * Explains a group's keep decision (spec V1-RUL-09): the generated default,
 * the trainer's override, protection, the minimum that won and the count.
 */
export function RuleInspector({
  group: g,
  name,
  rule,
  protectedByYou,
}: {
  group: Group;
  name: string;
  rule: SpeciesRule | undefined;
  protectedByYou: number;
}) {
  return (
    <section
      aria-label={`Why ${name} keeps ${g.keep}`}
      className="mt-2 rounded-xl bg-slate-50 p-3 text-sm dark:bg-slate-800/60"
    >
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
        <Row term="Default from your storage">{generated(g)}</Row>
        <Row term="Your rule">{own(g, rule)}</Row>
        <Row term="Protected">
          {protection(protectedByYou, g.protectedIds.length - protectedByYou)}
        </Row>
        <Row term="Minimum used">{used(g)}</Row>
        <Row term="Copies">{g.count}</Row>
      </dl>
    </section>
  );
}

function Row({ term, children }: { term: string; children: ReactNode }) {
  return (
    <div className="contents">
      <dt className="text-slate-500">{term}</dt>
      <dd className="tabular-nums">{children}</dd>
    </div>
  );
}
