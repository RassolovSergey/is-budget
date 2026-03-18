import { clearToken } from "./session";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

type RequestOptions = {
  method?: string;
  token?: string;
  body?: unknown;
};

export async function apiRequest<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: opts.method || "GET",
    headers: {
      "Content-Type": "application/json",
      ...(opts.token ? { Authorization: `Bearer ${opts.token}` } : {})
    },
    body: opts.body ? JSON.stringify(opts.body) : undefined,
    cache: "no-store"
  });

  if (res.status === 401 && !path.startsWith("/auth/")) {
    clearToken();
    if (typeof window !== "undefined" && window.location.pathname !== "/login") {
      const target = encodeURIComponent(window.location.pathname + window.location.search);
      window.location.href = `/login?next=${target}&reason=expired`;
    }
    throw new Error("Сессия истекла. Войдите заново.");
  }

  if (!res.ok) {
    let msg = `Request failed (${res.status})`;
    try {
      const payload = await res.json();
      if (payload.error) msg = payload.error;
    } catch {
      // keep default message
    }
    throw new Error(msg);
  }

  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export async function apiDownload(path: string, token?: string): Promise<Blob> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    cache: "no-store"
  });

  if (res.status === 401 && !path.startsWith("/auth/")) {
    clearToken();
    if (typeof window !== "undefined" && window.location.pathname !== "/login") {
      const target = encodeURIComponent(window.location.pathname + window.location.search);
      window.location.href = `/login?next=${target}&reason=expired`;
    }
    throw new Error("Сессия истекла. Войдите заново.");
  }

  if (!res.ok) {
    let msg = `Request failed (${res.status})`;
    try {
      const payload = await res.json();
      if (payload.error) msg = payload.error;
    } catch {
      // keep default message
    }
    throw new Error(msg);
  }

  return await res.blob();
}

export function apiAssetUrl(path: string): string {
  if (!path) return "";
  if (/^https?:\/\//i.test(path)) return path;
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return `${API_BASE}${normalized}`;
}
