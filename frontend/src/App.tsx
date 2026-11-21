import { Toaster } from "sonner";
import { Home } from "./home/Home";
import { HostStatus } from "./host/HostStatus";
import { OtherDevices } from "./host/OtherDevices";
import { StartAtLogin } from "./host/StartAtLogin";
import { SignedOut } from "./session/SignedOut";
import { useMe } from "./session/useSession";
import { card } from "./ui/styles";

export default function App() {
  return (
    <div className="min-h-dvh bg-soft text-ink">
      <main className="mx-auto flex max-w-md flex-col gap-4 px-4 py-8 sm:max-w-3xl">
        <header className="flex items-center gap-3">
          <img src="/favicon.svg" alt="" className="size-10" />
          <div>
            <h1 className="text-2xl font-bold tracking-tight">CatchKeep</h1>
            <p className="text-sm text-muted">A Pokémon GO storage helper.</p>
          </div>
        </header>
        <Session />
      </main>
      <Toaster position="top-center" richColors />
    </div>
  );
}

function Session() {
  const me = useMe();
  if (me.data) {
    return <Home displayName={me.data.displayName} />;
  }
  return (
    <>
      {!me.isPending && <SignedOut />}
      <section aria-label="Host" className={card}>
        <HostStatus />
        <OtherDevices />
        <StartAtLogin />
      </section>
    </>
  );
}
