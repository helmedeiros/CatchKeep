import { create } from "zustand";

interface Store {
  ticked: ReadonlySet<string>;
  set: (ids: string[], on: boolean) => void;
}

const useStore = create<Store>((set) => ({
  ticked: new Set(),
  set: (ids, on) =>
    set((s) => {
      const ticked = new Set(s.ticked);
      for (const id of ids) {
        if (on) ticked.add(id);
        else ticked.delete(id);
      }
      return { ticked };
    }),
}));

/**
 * The suggested transfers the trainer ticked off while transferring in the
 * game, remembered while they move around the app.
 */
export function useTicks() {
  return { ticked: useStore((s) => s.ticked), set: useStore((s) => s.set) };
}

/** Forgets every tick, for tests. */
export function resetTicks() {
  useStore.setState({ ticked: new Set() });
}
