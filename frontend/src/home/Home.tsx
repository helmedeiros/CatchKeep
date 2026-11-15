import { card, secondaryButton } from "../ui/styles";
import { useSignOut } from "../session/useSession";

export function Home({ displayName }: { displayName: string }) {
  const signOut = useSignOut();
  return (
    <section className={`${card} flex flex-col gap-3`}>
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Hi, {displayName}</h2>
        <button
          type="button"
          onClick={() => signOut.mutate()}
          className={secondaryButton}
        >
          Sign out
        </button>
      </div>
      <p className="text-sm text-slate-600 dark:text-slate-400">
        Your collection will appear here once you import it.
      </p>
    </section>
  );
}
