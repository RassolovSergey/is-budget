import { ApiError } from "./api";
import type { Locale } from "./i18n";

type InviteErrorPresentation = {
  inlineMessage: string;
  toastMessage?: string;
  suppressInline?: boolean;
};

function pick(locale: Locale, ru: string, en: string): string {
  return locale === "en" ? en : ru;
}

export function normalizeProjectInviteError(err: unknown, locale: Locale): InviteErrorPresentation {
  const raw = err instanceof Error ? err.message : "";
  const message = raw.trim();
  const status = err instanceof ApiError ? err.status : undefined;
  const code = err instanceof ApiError ? err.code : undefined;

  if (status === 409 && code === "conflict" && message === "invite already sent and pending confirmation") {
    const localized = pick(
      locale,
      "Приглашение уже отправлено и ожидает подтверждения.",
      "An invitation has already been sent and is awaiting confirmation."
    );
    return { inlineMessage: localized, toastMessage: localized, suppressInline: true };
  }

  if (status === 409 && code === "conflict" && message === "member already exists") {
    const localized = pick(
      locale,
      "Пользователь уже состоит в этом проекте.",
      "This user is already a member of the project."
    );
    return { inlineMessage: localized, toastMessage: localized, suppressInline: true };
  }

  if (status === 404 && code === "not_found" && message === "user not found") {
    return {
      inlineMessage: pick(
        locale,
        "Пользователь не найден. Проверьте email, UUID или никнейм.",
        "User not found. Check the email, UUID, or nickname."
      )
    };
  }

  if (status === 400 && code === "invalid_request") {
    if (message === "invalid role") {
      return {
        inlineMessage: pick(locale, "Указана недопустимая роль приглашения.", "The selected invite role is invalid.")
      };
    }
    if (message === "project administrator role is fixed for the creator and cannot be assigned") {
      return {
        inlineMessage: pick(
          locale,
          "Роль администратора нельзя назначить через приглашение.",
          "Administrator role cannot be assigned through an invite."
        )
      };
    }
  }

  return {
    inlineMessage: message || pick(locale, "Не удалось отправить приглашение.", "Failed to send invitation.")
  };
}
