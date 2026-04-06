"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";

import { apiRequest } from "../../../lib/api";
import { dispatchInvitesUpdated, dispatchProjectsUpdated } from "../../../lib/appEvents";
import { useAuthGuard } from "../../../lib/useAuthGuard";

export default function AcceptInvitePage() {
  const { token } = useParams<{ token: string }>();
  const router = useRouter();
  const { token: authToken, ready } = useAuthGuard();
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);

  useEffect(() => {
    if (!ready || !authToken || !token || done) return;

    let canceled = false;
    (async () => {
      try {
        const result = await apiRequest<{ project_id: string }>(`/project-invites/${token}/accept`, {
          method: "POST",
          token: authToken
        });
        if (canceled) return;
        setDone(true);
        dispatchInvitesUpdated();
        dispatchProjectsUpdated();
        router.replace(`/projects/${result.project_id}`);
      } catch (err) {
        if (canceled) return;
        setError((err as Error).message);
      }
    })();

    return () => {
      canceled = true;
    };
  }, [ready, authToken, token, done, router]);

  return (
    <div className="page" style={{ maxWidth: 760, paddingTop: 80 }}>
      <section className="card">
        <h1 className="title">Принятие приглашения</h1>
        {error ? (
          <p style={{ color: "#bc4749", margin: 0 }}>{error}</p>
        ) : (
          <p className="subtle" style={{ margin: 0 }}>
            Подключаем вас к проекту...
          </p>
        )}
      </section>
    </div>
  );
}
