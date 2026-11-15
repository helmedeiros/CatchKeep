import { Home } from "./home/Home";
import { HostStatus } from "./host/HostStatus";
import { SignedOut } from "./session/SignedOut";
import { useMe } from "./session/useSession";
import { card } from "./ui/styles";

export default function App() {
  return (
    <div className="min-h-dvh bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <main className="mx-auto flex max-w-md flex-col gap-4 px-4 py-8">
        <header className="flex items-center gap-3">
          <img src="/favicon.svg" alt="" className="size-10" />
          <div>
            <h1 className="text-2xl font-bold tracking-tight">CatchKeep</h1>
            <p className="text-sm text-slate-600 dark:text-slate-400">
              A Pokémon GO storage helper.
            </p>
          </div>
        </header>
        <Session />
        <section aria-label="Host" className={card}>
          <HostStatus />
        </section>
      </main>
    </div>
  );
}

function Session() {
  const me = useMe();
  if (me.isPending) {
    return null;
  }
  if (me.data) {
    return <Home displayName={me.data.displayName} />;
  }
  return <SignedOut />;
}
