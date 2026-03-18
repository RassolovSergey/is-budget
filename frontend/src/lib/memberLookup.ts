type MemberLookupResult = {
  email?: string;
  user_id?: string;
  nickname?: string;
};

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const nicknamePattern = /^[a-z0-9_]{3,32}$/;

export function parseMemberLookup(raw: string): MemberLookupResult | null {
  const value = raw.trim();
  if (!value) {
    return null;
  }

  if (value.startsWith("@")) {
    const nick = value.slice(1).trim().toLowerCase();
    if (!nicknamePattern.test(nick)) {
      return null;
    }
    return { nickname: nick };
  }

  if (uuidPattern.test(value)) {
    return { user_id: value };
  }

  const lower = value.toLowerCase();
  if (emailPattern.test(lower)) {
    return { email: lower };
  }

  if (nicknamePattern.test(lower)) {
    return { nickname: lower };
  }

  return null;
}
