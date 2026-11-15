import { CollectionView } from "../collection/CollectionView";
import { ConfirmFeatures } from "../collection/ConfirmFeatures";
import { ImportCard } from "../collection/ImportCard";
import { useCollection } from "../collection/useCollection";
import { KeepCard } from "../keep/KeepCard";
import { useSignOut } from "../session/useSession";
import { alert, card, secondaryButton } from "../ui/styles";

export function Home({ displayName }: { displayName: string }) {
  const signOut = useSignOut();
  const collection = useCollection();
  const pokemon = collection.data?.pokemon ?? [];

  return (
    <div className="flex flex-col gap-4">
      <section className={`${card} flex items-center justify-between`}>
        <h2 className="text-lg font-semibold">Hi, {displayName}</h2>
        <button
          type="button"
          onClick={() => signOut.mutate()}
          className={secondaryButton}
        >
          Sign out
        </button>
      </section>
      {collection.isError && (
        <p className={alert}>Couldn't load your collection.</p>
      )}
      {collection.isSuccess && (
        <>
          <ImportCard hasPokemon={pokemon.length > 0} />
          <ConfirmFeatures pokemon={pokemon} />
          {pokemon.length > 0 && <KeepCard />}
          <CollectionView pokemon={pokemon} />
        </>
      )}
    </div>
  );
}
