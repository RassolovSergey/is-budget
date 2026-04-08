type DisplayNameInput = {
  first_name?: string | null;
  last_name?: string | null;
  full_name?: string | null;
  nickname?: string | null;
  email?: string | null;
};

export function displayHumanName(person: DisplayNameInput | null | undefined, fallback = "Unknown"): string {
  if (!person) return fallback;

  const firstLast = `${person.first_name || ""} ${person.last_name || ""}`.trim();
  if (firstLast) return firstLast;

  const fullName = person.full_name?.trim();
  if (fullName) return fullName;

  const nickname = person.nickname?.trim();
  if (nickname) return nickname;

  const email = person.email?.trim();
  if (email) return email;

  return fallback;
}
