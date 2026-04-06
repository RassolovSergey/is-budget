"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { getToken } from "./session";

export function useAuthGuard() {
  const router = useRouter();
  const [token, setToken] = useState<string>("");
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const t = getToken();
    if (!t) {
      if (typeof window !== "undefined") {
        const target = encodeURIComponent(window.location.pathname + window.location.search);
        router.replace(`/login?next=${target}`);
        return;
      }
      router.replace("/login");
      return;
    }
    setToken(t);
    setReady(true);
  }, [router]);

  return { token, ready };
}
