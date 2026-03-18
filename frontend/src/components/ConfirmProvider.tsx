"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";

import { useLocale } from "./LocaleProvider";

type ConfirmTone = "default" | "danger";

type ConfirmOptions = {
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  tone?: ConfirmTone;
};

type ConfirmRequest = ConfirmOptions & {
  resolve: (accepted: boolean) => void;
};

type ConfirmFn = (options: ConfirmOptions) => Promise<boolean>;

const ConfirmContext = createContext<ConfirmFn | null>(null);

export function ConfirmProvider({ children }: { children: React.ReactNode }) {
  const { locale } = useLocale();
  const [request, setRequest] = useState<ConfirmRequest | null>(null);

  const closeDialog = useCallback((accepted: boolean) => {
    setRequest((current) => {
      if (current) {
        current.resolve(accepted);
      }
      return null;
    });
  }, []);

  const confirm = useCallback<ConfirmFn>((options) => {
    return new Promise<boolean>((resolve) => {
      setRequest({
        ...options,
        resolve
      });
    });
  }, []);

  useEffect(() => {
    if (!request) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        closeDialog(false);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [request, closeDialog]);

  const contextValue = useMemo(() => confirm, [confirm]);

  return (
    <ConfirmContext.Provider value={contextValue}>
      {children}
      {request && (
        <div className="confirm-overlay" role="presentation" onClick={() => closeDialog(false)}>
          <div
            className="confirm-dialog card"
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="confirm-title"
            aria-describedby="confirm-message"
            onClick={(event) => event.stopPropagation()}
          >
            <h3 id="confirm-title" className="confirm-title">{request.title}</h3>
            <p id="confirm-message" className="subtle confirm-message">{request.message}</p>
            <div className="confirm-actions">
              <button type="button" className="secondary" onClick={() => closeDialog(false)}>
                {request.cancelText ?? (locale === "en" ? "Cancel" : "Отмена")}
              </button>
              <button
                type="button"
                className={request.tone === "danger" ? "warn" : ""}
                onClick={() => closeDialog(true)}
              >
                {request.confirmText ?? (locale === "en" ? "Confirm" : "Подтвердить")}
              </button>
            </div>
          </div>
        </div>
      )}
    </ConfirmContext.Provider>
  );
}

export function useConfirm(): ConfirmFn {
  const context = useContext(ConfirmContext);
  if (!context) {
    throw new Error("useConfirm must be used within ConfirmProvider");
  }
  return context;
}
