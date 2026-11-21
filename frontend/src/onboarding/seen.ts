/** Where this device remembers that a profile has seen the introduction. */
export const onboardedKey = (profileId: string) =>
  `catchkeep:onboarded:${profileId}`;

export function hasSeen(profileId: string): boolean {
  try {
    return window.localStorage.getItem(onboardedKey(profileId)) === "1";
  } catch {
    return true;
  }
}

export function markSeen(profileId: string) {
  try {
    window.localStorage.setItem(onboardedKey(profileId), "1");
  } catch {
    // Private browsing: the introduction may show again next time.
  }
}
