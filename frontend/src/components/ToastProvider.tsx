"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";

type ToastTone = "info" | "success" | "warning" | "error";

type ToastItem = {
  id: number;
  message: string;
  tone: ToastTone;
};

type ShowToastInput = {
  message: string;
  tone?: ToastTone;
  durationMs?: number;
};

type ToastContextValue = {
  showToast: (input: ShowToastInput) => void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);
  const nextIdRef = useRef(1);
  const timeoutsRef = useRef<Map<number, number>>(new Map());

  const removeToast = useCallback((id: number) => {
    setItems((prev) => prev.filter((item) => item.id !== id));
    const timeout = timeoutsRef.current.get(id);
    if (timeout) {
      window.clearTimeout(timeout);
      timeoutsRef.current.delete(id);
    }
  }, []);

  const showToast = useCallback(
    ({ message, tone = "info", durationMs = 4200 }: ShowToastInput) => {
      const id = nextIdRef.current++;
      setItems((prev) => [...prev, { id, message, tone }]);
      const timeout = window.setTimeout(() => removeToast(id), durationMs);
      timeoutsRef.current.set(id, timeout);
    },
    [removeToast]
  );

  useEffect(() => {
    return () => {
      for (const timeout of timeoutsRef.current.values()) {
        window.clearTimeout(timeout);
      }
      timeoutsRef.current.clear();
    };
  }, []);

  const value = useMemo<ToastContextValue>(() => ({ showToast }), [showToast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-viewport" aria-live="polite" aria-atomic="true">
        {items.map((item) => (
          <div key={item.id} className={`toast-card toast-${item.tone}`}>
            <p>{item.message}</p>
            <button type="button" className="toast-close" onClick={() => removeToast(item.id)} aria-label="Close notification">
              ×
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return ctx;
}
