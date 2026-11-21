import { useMemo, useState } from "react";
import { toast } from "sonner";
import { ApiError } from "../api/errors";
import { PokemonRow } from "../collection/PokemonRow";
import type { Pokemon } from "../collection/useCollection";
import { Sprite } from "../ui/Sprite";
import {
  alert,
  card,
  input,
  primaryButton,
  secondaryButton,
} from "../ui/styles";
import {
  type Team,
  useCreateTeam,
  useDeleteTeam,
  useTeams,
  useUpdateTeam,
} from "./useTeams";

const MAX_MATCHES = 8;

const failure = (e: Error) =>
  e instanceof ApiError && e.code === "TEAM_NAME_TAKEN"
    ? "Another team already has that name."
    : e instanceof ApiError && e.code === "VALIDATION_FAILED"
      ? e.message
      : "Couldn't save the team.";

/** Named teams (V1-SET-03): every member is always kept. */
export function TeamsCard({ pokemon }: { pokemon: Pokemon[] }) {
  const teams = useTeams();
  const create = useCreateTeam();
  const [name, setName] = useState("");
  const [editing, setEditing] = useState<string | null>(null);
  const byId = useMemo(() => new Map(pokemon.map((p) => [p.id, p])), [pokemon]);

  return (
    <section aria-label="Your teams" className={`${card} flex flex-col gap-3`}>
      <div>
        <h3 className="font-semibold">Your teams</h3>
        <p className="text-sm text-muted">
          Every Pokémon on a team is always kept.
        </p>
      </div>
      {teams.isError && <p className={alert}>Couldn't load your teams.</p>}
      {teams.data && teams.data.length > 0 && (
        <ul className="flex flex-col divide-y divide-line">
          {teams.data.map((t) =>
            editing === t.id ? (
              <li key={t.id} className="py-2">
                <TeamEditor
                  team={t}
                  pokemon={pokemon}
                  byId={byId}
                  onDone={() => setEditing(null)}
                />
              </li>
            ) : (
              <li key={t.id} className="py-2">
                <TeamSummary
                  team={t}
                  byId={byId}
                  onEdit={() => setEditing(t.id)}
                />
              </li>
            ),
          )}
        </ul>
      )}
      <form
        className="flex flex-col gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate(
            { name, memberIds: [] },
            {
              onSuccess: (team) => {
                setName("");
                setEditing(team.id);
              },
              onError: (err) => toast.error(failure(err)),
            },
          );
        }}
      >
        <label className="sr-only" htmlFor="new-team">
          New team name
        </label>
        <input
          id="new-team"
          className={input}
          placeholder="New team, e.g. Raid Kyurem"
          maxLength={40}
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button
          type="submit"
          className={primaryButton}
          disabled={name.trim() === "" || create.isPending}
        >
          Create team
        </button>
      </form>
    </section>
  );
}

function TeamSummary({
  team: t,
  byId,
  onEdit,
}: {
  team: Team;
  byId: Map<string, Pokemon>;
  onEdit: () => void;
}) {
  const remove = useDeleteTeam();
  const [confirming, setConfirming] = useState(false);
  return (
    <div
      role="group"
      aria-label={t.name}
      className="flex flex-wrap items-center gap-3"
    >
      <div className="min-w-0 flex-1">
        <p className="font-medium">{t.name}</p>
        <p className="text-sm text-muted">{t.memberIds.length} Pokémon</p>
      </div>
      <div className="flex -space-x-2">
        {t.memberIds.slice(0, 6).map((id) => {
          const p = byId.get(id);
          return (
            p && (
              <Sprite
                key={id}
                size="sm"
                species={p.species}
                dexNumber={p.dexNumber}
                shiny={p.attributes.shiny === "yes"}
              />
            )
          );
        })}
      </div>
      {confirming ? (
        <div className="flex items-center gap-2 text-sm">
          <span>Delete {t.name}?</span>
          <button
            type="button"
            className={secondaryButton}
            disabled={remove.isPending}
            onClick={() =>
              remove.mutate(t.id, {
                onSuccess: () => toast.success(`Deleted ${t.name}.`),
                onError: () => toast.error(`Couldn't delete ${t.name}.`),
              })
            }
          >
            Yes, delete
          </button>
          <button
            type="button"
            className={secondaryButton}
            onClick={() => setConfirming(false)}
          >
            Keep it
          </button>
        </div>
      ) : (
        <div className="flex gap-2">
          <button type="button" className={secondaryButton} onClick={onEdit}>
            Edit
          </button>
          <button
            type="button"
            className={secondaryButton}
            onClick={() => setConfirming(true)}
          >
            Delete
          </button>
        </div>
      )}
    </div>
  );
}

function TeamEditor({
  team,
  pokemon,
  byId,
  onDone,
}: {
  team: Team;
  pokemon: Pokemon[];
  byId: Map<string, Pokemon>;
  onDone: () => void;
}) {
  const update = useUpdateTeam();
  const [name, setName] = useState(team.name);
  const [members, setMembers] = useState(team.memberIds);
  const [query, setQuery] = useState("");
  const q = query.trim().toLowerCase();
  const matches =
    q === ""
      ? []
      : pokemon
          .filter(
            (p) =>
              !members.includes(p.id) && p.species.toLowerCase().includes(q),
          )
          .slice(0, MAX_MATCHES);

  return (
    <form
      aria-label={`Edit ${team.name}`}
      className="flex flex-col gap-3"
      onSubmit={(e) => {
        e.preventDefault();
        update.mutate(
          { id: team.id, name, memberIds: members },
          {
            onSuccess: () => {
              toast.success(`Saved ${name.trim()}.`);
              onDone();
            },
            onError: (err) => toast.error(failure(err)),
          },
        );
      }}
    >
      <label className="flex flex-col gap-1 text-sm">
        <span className="text-muted">Team name</span>
        <input
          className={input}
          maxLength={40}
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </label>
      {members.length > 0 && (
        <ul aria-label="Members" className="flex flex-col divide-y divide-line">
          {members.map((id) => {
            const p = byId.get(id);
            return (
              p && (
                <li key={id} className="py-2">
                  <PokemonRow
                    pokemon={p}
                    action={
                      <button
                        type="button"
                        className={secondaryButton}
                        aria-label={`Remove ${p.species} CP ${p.cp ?? "?"}`}
                        onClick={() =>
                          setMembers((m) => m.filter((x) => x !== id))
                        }
                      >
                        Remove
                      </button>
                    }
                  />
                </li>
              )
            );
          })}
        </ul>
      )}
      <label className="flex flex-col gap-1 text-sm">
        <span className="text-muted">Add Pokémon</span>
        <input
          type="search"
          className={input}
          placeholder="Search by species"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </label>
      {matches.length > 0 && (
        <ul aria-label="Matches" className="flex flex-col divide-y divide-line">
          {matches.map((p) => (
            <li key={p.id} className="py-2">
              <PokemonRow
                pokemon={p}
                action={
                  <button
                    type="button"
                    className={secondaryButton}
                    aria-label={`Add ${p.species} CP ${p.cp ?? "?"}`}
                    onClick={() => setMembers((m) => [...m, p.id])}
                  >
                    Add
                  </button>
                }
              />
            </li>
          ))}
        </ul>
      )}
      <div className="flex gap-2">
        <button
          type="submit"
          className={primaryButton}
          disabled={name.trim() === "" || update.isPending}
        >
          Save team
        </button>
        <button type="button" className={secondaryButton} onClick={onDone}>
          Cancel
        </button>
      </div>
    </form>
  );
}
