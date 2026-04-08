"use client";

import { useEffect, useRef } from "react";

type Options = {
  enabled?: boolean;
  intervalMs?: number;
  eventNames?: string[];
};

export function useLiveRefresh(
  refresh: () => void | Promise<void>,
  { enabled = true, intervalMs = 8000, eventNames = [] }: Options = {}
) {
  const refreshRef = useRef(refresh);
  const eventKey = eventNames.join("|");

  useEffect(() => {
    refreshRef.current = refresh;
  }, [refresh]);

  useEffect(() => {
    if (!enabled || typeof window === "undefined") return;

    const tick = () => {
      if (document.visibilityState === "hidden") return;
      void refreshRef.current();
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        tick();
      }
    };

    const intervalId = window.setInterval(tick, intervalMs);
    document.addEventListener("visibilitychange", handleVisibilityChange);
    window.addEventListener("focus", tick);
    eventNames.forEach((eventName) => window.addEventListener(eventName, tick));

    return () => {
      window.clearInterval(intervalId);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.removeEventListener("focus", tick);
      eventNames.forEach((eventName) => window.removeEventListener(eventName, tick));
    };
  }, [enabled, eventKey, intervalMs]);
}
