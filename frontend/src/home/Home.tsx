import { type ReactNode, useState } from "react";
import { Nav } from "../app/Nav";
import { navigate, usePath } from "../app/router";
import { CollectionView } from "../collection/CollectionView";
import { ConfirmFeatures } from "../collection/ConfirmFeatures";
import { ImportCard } from "../collection/ImportCard";
import { countUnconfirmed } from "../collection/pokemon";
import { type Pokemon, useCollection } from "../collection/useCollection";
import { HostStatus } from "../host/HostStatus";
import { OtherDevices } from "../host/OtherDevices";
import { StartAtLogin } from "../host/StartAtLogin";
import { KeepCard } from "../keep/KeepCard";
import { MakeRoom } from "../plan/MakeRoom";
import { NextStep } from "./NextStep";
import { Onboarding } from "../onboarding/Onboarding";
import { hasSeen, markSeen } from "../onboarding/seen";
import { useSignOut } from "../session/useSession";
import { TeamsCard } from "../teams/TeamsCard";
import { alert, card, secondaryButton } from "../ui/styles";

/** The signed-in app: five places behind the main menu. */
export function Home({
  displayName,
  profileId,
}: {
  displayName: string;
  profileId: string;
}) {
  const [intro, setIntro] = useState(() => !hasSeen(profileId));
  const collection = useCollection();
  const path = usePath();
  const pokemon = collection.data?.pokemon ?? [];

  let place: ReactNode = null;
  if (collection.isSuccess) {
    switch (path) {
      case "/collection":
        place = (
          <Place title="Collection">
            <CollectionView pokemon={pokemon} />
          </Place>
        );
        break;
      case "/make-room":
        place = (
          <Place title="Make room">
            {pokemon.length > 0 ? (
              <MakeRoom pokemon={pokemon} />
            ) : (
              <ImportFirst />
            )}
          </Place>
        );
        break;
      case "/rules":
        place = (
          <Place title="Rules">
            {pokemon.length > 0 ? (
              <>
                <KeepCard pokemon={pokemon} />
                <TeamsCard pokemon={pokemon} />
              </>
            ) : (
              <ImportFirst />
            )}
          </Place>
        );
        break;
      case "/more":
        place = (
          <More
            hasPokemon={pokemon.length > 0}
            onIntro={() => setIntro(true)}
          />
        );
        break;
      default:
        place = (
          <Place title={`Hi, ${displayName}`}>
            <NextStep pokemon={pokemon} />
            {pokemon.length === 0 && <ImportCard hasPokemon={false} />}
            {countUnconfirmed(pokemon) > 0 && (
              <ConfirmFeatures pokemon={pokemon} />
            )}
            {pokemon.length > 0 && <Shortcuts pokemon={pokemon} />}
          </Place>
        );
    }
  }

  return (
    <div className="flex flex-col gap-4 pb-24 sm:pb-0">
      <Nav />
      <Onboarding
        open={intro}
        onClose={() => {
          markSeen(profileId);
          setIntro(false);
        }}
      />
      {collection.isError && (
        <p className={alert}>Couldn't load your collection.</p>
      )}
      {place}
    </div>
  );
}

function Place({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-4">
      <h2 className="px-1 text-2xl font-bold tracking-tight">{title}</h2>
      {children}
    </div>
  );
}

function ImportFirst() {
  return (
    <p className={`${card} text-sm text-muted`}>
      Bring in your collection first, from Home.
    </p>
  );
}

function Shortcuts({ pokemon }: { pokemon: Pokemon[] }) {
  const items = [
    {
      to: "/make-room",
      title: "Make room",
      text: "See what you can transfer before your next big catch.",
    },
    {
      to: "/collection",
      title: "Collection",
      text: `Browse all ${pokemon.length} Pokémon.`,
    },
    {
      to: "/rules",
      title: "Rules",
      text: "Storage, goals, your own rules and teams.",
    },
  ];
  return (
    <ul className="grid gap-3 sm:grid-cols-3">
      {items.map((it) => (
        <li key={it.to}>
          <a
            href={it.to}
            onClick={(e) => {
              e.preventDefault();
              navigate(it.to);
            }}
            className={`${card} flex h-full flex-col gap-1 hover:ring-2 hover:ring-accent`}
          >
            <span className="font-semibold">{it.title}</span>
            <span className="text-sm text-muted">{it.text}</span>
          </a>
        </li>
      ))}
    </ul>
  );
}

function More({
  hasPokemon,
  onIntro,
}: {
  hasPokemon: boolean;
  onIntro: () => void;
}) {
  const signOut = useSignOut();
  return (
    <Place title="More">
      {hasPokemon && <ImportCard hasPokemon />}
      <section aria-label="This computer" className={card}>
        <HostStatus />
        <OtherDevices />
        <StartAtLogin />
      </section>
      <button
        type="button"
        onClick={onIntro}
        className={`${secondaryButton} self-start`}
      >
        Show the introduction again
      </button>
      <button
        type="button"
        onClick={() => signOut.mutate()}
        className={`${secondaryButton} self-start`}
      >
        Sign out
      </button>
    </Place>
  );
}
