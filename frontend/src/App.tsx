import { HostStatus } from "./host/HostStatus";

export default function App() {
  return (
    <div className="min-h-dvh bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <main className="mx-auto flex max-w-md flex-col gap-6 px-4 py-10">
        <header className="flex items-center gap-3">
          <img src="/favicon.svg" alt="" className="size-10" />
          <div>
            <h1 className="text-2xl font-bold tracking-tight">CatchKeep</h1>
            <p className="text-sm text-slate-600 dark:text-slate-400">
              A Pokémon GO storage helper.
            </p>
          </div>
        </header>
        <section
          aria-label="Host"
          className="rounded-2xl bg-white p-4 shadow-sm ring-1 ring-slate-200 dark:bg-slate-900 dark:ring-slate-800"
        >
          <HostStatus />
        </section>
      </main>
    </div>
  );
}
