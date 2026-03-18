"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";

import { AppShell } from "../../../../components/AppShell";
import { useLocale } from "../../../../components/LocaleProvider";
import { ProjectNav } from "../../../../components/ProjectNav";
import { apiRequest } from "../../../../lib/api";
import type { Project } from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";

export default function LegacyHistoricalCampaignsRedirectPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { locale } = useLocale();
  const { token, ready } = useAuthGuard();
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!ready || !token) return;

    let cancelled = false;
    (async () => {
      try {
        const projectResp = await apiRequest<Project>(`/projects/${id}`, { token });
        if (cancelled) return;
        setProject(projectResp);
        const category = encodeURIComponent(projectResp.category || tr("Общее", "General"));
        router.replace(`/historical-campaigns?category=${category}`);
      } catch (err) {
        if (!cancelled) {
          setError((err as Error).message);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [id, ready, router, token]);

  if (!ready) return null;

  return (
    <AppShell title={project?.name || tr("История кампаний", "Campaign History")}>
      {project && <ProjectNav projectId={project.id} />}
      <section className="card" style={{ maxWidth: 720 }}>
        <h2 className="title">{tr("История кампаний перенесена", "Campaign history has moved")}</h2>
        <p className="subtle" style={{ marginTop: 0 }}>
          {tr(
            "Мы перенесли ручной ввод истории в общий раздел пользователя. Если редирект не сработал автоматически, откройте раздел «История» в верхнем меню.",
            "Manual campaign history entry is now in the global user section. If automatic redirect did not work, open the History section in the top menu."
          )}
        </p>
        {error && <p style={{ color: "#bc4749", margin: 0 }}>{error}</p>}
      </section>
    </AppShell>
  );
}
