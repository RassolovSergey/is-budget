"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

import { getToken } from "../lib/session";

export default function HomePage() {
  const router = useRouter();

  useEffect(() => {
    router.replace(getToken() ? "/projects" : "/login");
  }, [router]);

  return null;
}
