/**
 * Pokémon renders are not bundled: they load from the public PokeAPI sprite
 * collection by National Pokédex number, falling back to the initial.
 */
const BASE =
  "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/other/home";

export function spriteUrl(dexNumber: number, shiny = false): string {
  return `${BASE}/${shiny ? "shiny/" : ""}${dexNumber}.png`;
}
