const TOKEN_KEY = "is_budget_token";

export function getToken(): string {
  if (typeof window === "undefined") return "";
  const raw = window.localStorage.getItem(TOKEN_KEY);
  if (!raw || raw === "null" || raw === "undefined") return "";
  return raw.trim();
}

export function setToken(token: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(TOKEN_KEY);
}
