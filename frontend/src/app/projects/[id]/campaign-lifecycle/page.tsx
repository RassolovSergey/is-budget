"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

import { AppShell } from "../../../../components/AppShell";
import { useLocale } from "../../../../components/LocaleProvider";
import { ProjectNav } from "../../../../components/ProjectNav";
import { ApiError, apiRequest, apiUpload } from "../../../../lib/api";
import type {
  CampaignConfig,
  CampaignLifecycleStage,
  CampaignStageSnapshot,
  ImportRunDetail,
  ImportRunSummary,
  Project
} from "../../../../lib/types";
import { useAuthGuard } from "../../../../lib/useAuthGuard";
import {
  CampaignLifecycleConfigPanel,
  CampaignStageSnapshotPanel,
  KickstarterImportPanel,
  stageLabel,
  type CampaignConfigFormState,
  type CampaignSnapshotFormState,
  type KickstarterImportFormState
} from "./LifecycleSections";

function isoToLocalInput(value?: string | null): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function formatLifecycleError(err: unknown, locale: "ru" | "en"): string {
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);
  if (err instanceof ApiError) {
    if (err.status === 403) {
      return tr("У вас нет прав для этого действия в жизненном цикле кампании.", "You do not have permission for this lifecycle action.");
    }
    if (err.status === 404) {
      return tr("Снимок стадии не найден.", "The lifecycle snapshot was not found.");
    }
    if (err.code === "invalid_csv") {
      return tr("CSV не прошёл проверку импорта Kickstarter.", "The CSV did not pass Kickstarter import validation.");
    }
    if (err.status === 409 || err.code === "conflict") {
      return tr("Импорт затронет существующие даты дохода. Включите замену существующих значений, если это ожидаемо.", "The import would overwrite existing income dates. Enable replace existing if that is expected.");
    }
  }
  return (err as Error)?.message || tr("Не удалось выполнить действие в жизненном цикле кампании.", "Failed to perform the lifecycle action.");
}

export default function CampaignLifecyclePage() {
  const { id } = useParams<{ id: string }>();
  const { locale } = useLocale();
  const { token, ready } = useAuthGuard();
  const appLocale = locale === "en" ? "en" : "ru";
  const tr = (ru: string, en: string) => (locale === "en" ? en : ru);

  const [project, setProject] = useState<Project | null>(null);
  const [config, setConfig] = useState<CampaignConfig | null>(null);
  const [snapshots, setSnapshots] = useState<CampaignStageSnapshot[]>([]);
  const [selectedSnapshot, setSelectedSnapshot] = useState<CampaignStageSnapshot | null>(null);
  const [imports, setImports] = useState<ImportRunSummary[]>([]);
  const [selectedImport, setSelectedImport] = useState<ImportRunDetail | null>(null);
  const [configForm, setConfigForm] = useState<CampaignConfigFormState>({
    current_stage: "pre_launch",
    launch_at: "",
    late_pledges_enabled: false,
    pledge_over_time_enabled: false,
    pledge_manager_enabled: false,
    digital_only_fulfillment: false
  });
  const [snapshotForm, setSnapshotForm] = useState<CampaignSnapshotFormState>({
    stage: "pre_launch",
    followers_count: "",
    notes: "",
    gross_amount: ""
  });
  const [importForm, setImportForm] = useState<KickstarterImportFormState>({
    replace_existing: false
  });
  const [loading, setLoading] = useState(true);
  const [busyKey, setBusyKey] = useState("");
  const [error, setError] = useState("");
  const [configError, setConfigError] = useState("");
  const [snapshotError, setSnapshotError] = useState("");
  const [importError, setImportError] = useState("");
  const [notice, setNotice] = useState("");

  const canManage = project?.current_user_role === "PM";
  const latestSuccessfulImport = imports.find((item) => item.status === "success") || null;
  const shouldPromptPreLaunchSnapshot =
    config?.current_stage === "pre_launch" && !snapshots.some((snapshot) => snapshot.stage === "pre_launch");

  async function load() {
    if (!token) return;
    try {
      setLoading(true);
      setError("");
      const [projectResp, configResp, snapshotsResp, importsResp] = await Promise.all([
        apiRequest<Project>(`/projects/${id}`, { token }),
        apiRequest<CampaignConfig>(`/projects/${id}/campaign-config`, { token }),
        apiRequest<CampaignStageSnapshot[]>(`/projects/${id}/campaign-stage-snapshots`, { token }),
        apiRequest<ImportRunSummary[]>(`/projects/${id}/imports`, { token })
      ]);
      setProject(projectResp);
      setConfig(configResp);
      setSnapshots(snapshotsResp);
      setImports(importsResp);
      setConfigForm({
        current_stage: configResp.current_stage,
        launch_at: isoToLocalInput(configResp.launch_at),
        late_pledges_enabled: configResp.late_pledges_enabled,
        pledge_over_time_enabled: configResp.pledge_over_time_enabled,
        pledge_manager_enabled: configResp.pledge_manager_enabled,
        digital_only_fulfillment: configResp.digital_only_fulfillment
      });
      setSnapshotForm((current) => ({
        ...current,
        stage: configResp.current_stage
      }));
      if (selectedSnapshot) {
        const refreshed = snapshotsResp.find((item) => item.id === selectedSnapshot.id);
        setSelectedSnapshot(refreshed || null);
      }
      if (selectedImport) {
        const refreshed = importsResp.find((item) => item.id === selectedImport.run.id);
        if (!refreshed) {
          setSelectedImport(null);
        }
      }
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (ready) {
      void load()
    }
  }, [ready, id, token]);

  async function saveConfig() {
    if (!token || !canManage) return;
    try {
      setBusyKey("config");
      setConfigError("");
      setNotice("");
      const updated = await apiRequest<CampaignConfig>(`/projects/${id}/campaign-config`, {
        method: "PUT",
        token,
        body: {
          current_stage: configForm.current_stage,
          launch_at: configForm.launch_at || null,
          late_pledges_enabled: configForm.late_pledges_enabled,
          pledge_over_time_enabled: configForm.pledge_over_time_enabled,
          pledge_manager_enabled: configForm.pledge_manager_enabled,
          digital_only_fulfillment: configForm.digital_only_fulfillment
        }
      });
      setConfig(updated);
      setNotice(tr("Настройки жизненного цикла сохранены.", "Lifecycle config saved."));
      await load();
    } catch (err) {
      setConfigError(formatLifecycleError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  async function saveSnapshot() {
    if (!token || !canManage) return;
    const followers = snapshotForm.followers_count.trim() ? Number(snapshotForm.followers_count) : null;
    if (followers != null && (!Number.isFinite(followers) || followers < 0)) {
      setSnapshotError(tr("Число подписчиков должно быть числом 0 или больше.", "Followers count must be a number 0 or greater."));
      return;
    }
    const gross = snapshotForm.gross_amount.trim() ? Number(snapshotForm.gross_amount) : null;
    if (gross != null && (!Number.isFinite(gross) || gross < 0)) {
      setSnapshotError(tr("Переопределение валового дохода должно быть числом 0 или больше.", "Gross override must be a number 0 or greater."));
      return;
    }

    try {
      setBusyKey("snapshot");
      setSnapshotError("");
      setNotice("");
      const created = await apiRequest<CampaignStageSnapshot>(`/projects/${id}/campaign-stage-snapshots`, {
        method: "POST",
        token,
        body: {
          stage: snapshotForm.stage,
          followers_count: followers,
          notes: snapshotForm.notes.trim() || undefined,
          gross_amount: gross
        }
      });
      setSelectedSnapshot(created);
      setNotice(tr("Снимок стадии сохранён.", "Stage snapshot saved."));
      setSnapshotForm((current) => ({
        ...current,
        followers_count: "",
        notes: "",
        gross_amount: ""
      }));
      await load();
    } catch (err) {
      setSnapshotError(formatLifecycleError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  async function openSnapshot(snapshotID: string) {
    if (!token) return;
    try {
      setBusyKey(`detail:${snapshotID}`);
      const detail = await apiRequest<CampaignStageSnapshot>(`/campaign-stage-snapshots/${snapshotID}`, { token });
      setSelectedSnapshot(detail);
    } catch (err) {
      setError(formatLifecycleError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  async function importKickstarterCSV(file: File) {
    if (!token || !canManage) return;
    try {
      setBusyKey("import");
      setImportError("");
      setNotice("");
      const formData = new FormData();
      formData.append("file", file);
      formData.append("replace_existing", importForm.replace_existing ? "true" : "false");
      const created = await apiUpload<ImportRunDetail>(`/projects/${id}/imports/kickstarter`, {
        token,
        body: formData
      });
      setSelectedImport(created);
      setNotice(tr("Импорт Kickstarter CSV завершён.", "Kickstarter CSV import finished."));
      await load();
    } catch (err) {
      setImportError(formatLifecycleError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  async function openImport(runID: string) {
    if (!token) return;
    try {
      setBusyKey(`import:${runID}`);
      const detail = await apiRequest<ImportRunDetail>(`/imports/${runID}`, { token });
      setSelectedImport(detail);
    } catch (err) {
      setError(formatLifecycleError(err, appLocale));
    } finally {
      setBusyKey("");
    }
  }

  if (!ready || loading || !project || !config) return null;

  return (
    <AppShell title={project.name}>
      <ProjectNav projectId={id} />

      <section className="card workflow-hero-card finance-pipeline-hero-card">
        <div>
          <span className="shell-eyebrow">{tr("Жизненный цикл кампании", "Campaign lifecycle")}</span>
          <h1 className="title">{tr("Операционный контроль стадий кампании", "Operations control for campaign stages")}</h1>
          <p>
            {tr(
              "Отслеживайте текущую стадию жизненного цикла Kickstarter, момент запуска и фиксируйте контрольные точки до и после кампании.",
              "Track the current Kickstarter lifecycle stage, launch milestone, and freeze pre-launch / post-launch checkpoints."
            )}
          </p>
        </div>
        <div className="lifecycle-stage-hero">
          <span className="workflow-status-badge status-in_progress">{stageLabel(config.current_stage, appLocale)}</span>
          <p className="workflow-inline-note">
            {config.launch_at
              ? tr(`Момент запуска: ${new Date(config.launch_at).toLocaleString("ru-RU")}`, `Launch milestone: ${new Date(config.launch_at).toLocaleString("en-US")}`)
              : tr("Момент запуска пока не задан.", "Launch milestone is not set yet.")}
          </p>
        </div>
      </section>

      {error ? <p className="workflow-inline-error">{error}</p> : null}
      {notice ? <p className="workflow-inline-success">{notice}</p> : null}

      <section className="card finance-ops-flow-card">
        <div>
          <span className="shell-eyebrow">{tr("Операционный маршрут", "Operations path")}</span>
          <h2 className="title">{tr("Что делать после обновления кампании", "What to do after updating the campaign")}</h2>
        </div>
        <div className="finance-ops-flow-grid">
          <a href="#kickstarter-import" className="finance-ops-step">
            <strong>{tr("1. Импортировать данные Kickstarter", "1. Import Kickstarter data")}</strong>
            <p>{tr("Загрузите CSV и обновите доходы проекта по датам.", "Upload the CSV and refresh project income by date.")}</p>
          </a>
          <Link href={`/projects/${id}/finance-pipeline`} className="finance-ops-step">
            <strong>{tr("2. Проверить финансовый конвейер", "2. Review the finance pipeline")}</strong>
            <p>{tr("Убедитесь, что уровни налогов, расходов и распределения дают правильный командный пул.", "Make sure taxes, expenses, and distributions produce the correct team pool.")}</p>
          </Link>
          <Link href={`/projects/${id}/my-income`} className="finance-ops-step">
            <strong>{tr("3. Проверить прогнозы и выплаты", "3. Review forecasts and payouts")}</strong>
            <p>{tr("Сравните личные и командные суммы после обновления данных и снимков.", "Compare personal and team amounts after refreshing data and snapshots.")}</p>
          </Link>
        </div>
      </section>

      {shouldPromptPreLaunchSnapshot ? (
        <section className="card lifecycle-next-step-card">
          <div>
            <span className="shell-eyebrow">{tr("Перед запуском", "Before launch")}</span>
            <h2 className="title">{tr("Сохраните снимок стадии до перехода в активную кампанию", "Save a stage snapshot before going live")}</h2>
            <p className="subtle">
              {tr(
                "Если вы ещё не зафиксировали число подписчиков и ключевые метрики до запуска, сделайте это в разделе «Снимок стадии». Потом будет проще сравнивать ожидания и фактический старт кампании.",
                "If you have not frozen follower count and key metrics before launch yet, do it in the stage snapshot section. It makes it much easier to compare expectations with the actual campaign start."
              )}
            </p>
          </div>
          <a href="#stage-snapshot" className="button-like">
            {tr("Перейти к снимку стадии", "Go to stage snapshot")}
          </a>
        </section>
      ) : null}

      <div className="lifecycle-page-stack">
        <CampaignLifecycleConfigPanel
          locale={appLocale}
          canManage={Boolean(canManage)}
          form={configForm}
          busy={busyKey === "config"}
          error={configError}
          onChange={setConfigForm}
          onSubmit={() => void saveConfig()}
        />

        <CampaignStageSnapshotPanel
          locale={appLocale}
          canManage={Boolean(canManage)}
          currencyCode={project.currency_code}
          form={snapshotForm}
          busy={busyKey === "snapshot"}
          error={snapshotError}
          snapshots={snapshots}
          selectedSnapshot={selectedSnapshot}
          onChange={setSnapshotForm}
          onSubmit={() => void saveSnapshot()}
          onOpen={(snapshotID) => void openSnapshot(snapshotID)}
        />

        <KickstarterImportPanel
          locale={appLocale}
          canManage={Boolean(canManage)}
          currencyCode={project.currency_code}
          form={importForm}
          busy={busyKey === "import"}
          error={importError}
          imports={imports}
          selectedImport={selectedImport}
          onChange={setImportForm}
          onPickFile={(file) => void importKickstarterCSV(file)}
          onOpen={(runID) => void openImport(runID)}
        />

        {(selectedImport?.run.status === "success" || latestSuccessfulImport) && (
          <section className="card lifecycle-next-step-card">
            <div>
              <span className="shell-eyebrow">{tr("После импорта", "After import")}</span>
              <h2 className="title">{tr("Что проверить дальше", "What to review next")}</h2>
              <p className="subtle">
                {tr(
                  "Доходы проекта уже обновлены. Теперь проверьте, как эти данные меняют денежный поток и прогноз выплат.",
                  "Project income has already been refreshed. Now verify how the new data changes the money flow and payout forecast."
                )}
              </p>
              <p className="subtle" style={{ margin: 0 }}>
                {tr("Последний успешный импорт", "Latest successful import")}:{" "}
                <strong>{(selectedImport?.run.status === "success" ? selectedImport.run.file_name : latestSuccessfulImport?.file_name) || "—"}</strong>
              </p>
            </div>
            <div className="lifecycle-next-step-actions">
              <Link href={`/projects/${id}/finance-pipeline`} className="button-like">
                {tr("Проверить финансовый конвейер", "Review Finance Pipeline")}
              </Link>
              <Link href={`/projects/${id}/my-income`} className="button-like secondary">
                {tr("Проверить прогнозы и выплаты", "Review forecasts and payouts")}
              </Link>
            </div>
          </section>
        )}
      </div>
    </AppShell>
  );
}
