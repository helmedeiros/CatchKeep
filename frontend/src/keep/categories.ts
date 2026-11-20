import type { components } from "../api/schema.gen";

export type Category = components["schemas"]["SpeciesCategory"];

/** Category names as they read mid-sentence: "each mythical species". */
export const CATEGORY_NAMES: Record<Category, string> = {
  legendary: "legendary",
  mythical: "mythical",
  "ultra-beast": "Ultra Beast",
};

/** Category names as titles: "Keep rule for Mythical". */
export const CATEGORY_TITLES: Record<Category, string> = {
  legendary: "Legendary",
  mythical: "Mythical",
  "ultra-beast": "Ultra Beasts",
};

export const CATEGORIES = Object.keys(CATEGORY_NAMES) as Category[];
