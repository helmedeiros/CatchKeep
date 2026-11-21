import { create } from "zustand";
import { type Filters, NO_FILTERS } from "./filters";

interface Store {
  byList: Record<string, Filters>;
  update: (list: string, change: Partial<Filters>) => void;
}

const useStore = create<Store>((set) => ({
  byList: {},
  update: (list, change) =>
    set((s) => ({
      byList: {
        ...s.byList,
        [list]: { ...(s.byList[list] ?? NO_FILTERS), ...change },
      },
    })),
}));

/** A list's search and filters, remembered while the trainer moves around. */
export function useFilters(list: string, defaults: Filters = NO_FILTERS) {
  const filters = useStore((s) => s.byList[list] ?? defaults);
  const update = useStore((s) => s.update);
  return {
    filters,
    update: (change: Partial<Filters>) =>
      update(list, {
        ...(useStore.getState().byList[list] ?? defaults),
        ...change,
      }),
    reset: () => update(list, defaults),
  };
}

/** Forgets every list's filters, for tests. */
export function resetAllFilters() {
  useStore.setState({ byList: {} });
}
