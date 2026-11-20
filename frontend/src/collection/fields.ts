import type { components } from "../api/schema.gen";

export type MappableField = components["schemas"]["MappableField"];

/** What each field is called when matching columns. */
export const FIELD_LABELS: Record<MappableField, string> = {
  species: "Species",
  dex_number: "Pokédex number",
  form: "Form",
  gender: "Gender",
  cp: "CP",
  hp: "HP",
  level: "Level",
  iv_attack: "Attack IV",
  iv_defense: "Defense IV",
  iv_stamina: "Stamina IV",
  catch_date: "Catch date",
  shiny: "Shiny (yes/no)",
  lucky: "Lucky (yes/no)",
  shadow: "Shadow (yes/no)",
  purified: "Purified (yes/no)",
  costume: "Costume (yes/no)",
  background: "Background (yes/no)",
  dynamax: "Dynamax (yes/no)",
  gigantamax: "Gigantamax (yes/no)",
  in_game_favorite: "In-game favorite (yes/no)",
};

export const FIELDS = Object.keys(FIELD_LABELS) as MappableField[];
