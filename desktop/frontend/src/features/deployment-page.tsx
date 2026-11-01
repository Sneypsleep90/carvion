import { useCallback, useEffect, useMemo, useState } from "react";
import { Activity, Ban, CheckCircle2, Clock3, LoaderCircle, PackageCheck, Play, RefreshCw, ShieldAlert, ShieldCheck, XCircle } from "lucide-react";
import {
  DeploymentCheckStatus,
  DeploymentMode,
  DeploymentState,
} from "../../bindings/github.com/Sneypsleep90/carvion/internal/core/domain/models";
import type { DeploymentCheck, DeploymentJob, DeploymentJobSummary, SimulatorProfile } from "../../bindings/github.com/Sneypsleep90/carvion/internal/core/domain/models";
import type { DesktopBackend, LocalizationProjectSummary } from "../lib/backend";
import { errorMessage } from "../lib/utils";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Skeleton } from "../components/ui/skeleton";

type DeploymentPageProps = Readonly<{ backend: DesktopBackend }>;

export function DeploymentPage({ backend }: DeploymentPageProps) {
  const [projects, setProjects] = useState<ReadonlyArray<LocalizationProjectSummary>>([]);
  const [profiles, setProfiles] = useState<ReadonlyArray<SimulatorProfile>>([]);
  const [history, setHistory] = useState<ReadonlyArray<DeploymentJobSummary>>([]);
  const [selectedProjectID, setSelectedProjectID] = useState("");
  const [selectedProfileID, setSelectedProfileID] = useState("");
  const [job, setJob] = useState<DeploymentJob | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [canceling, setCanceling] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const supportedSyntheticProfiles = useMemo(
    () => profiles.filter((profile) => profile.synthetic && profile.expectedCompatibility === "supported"),
    [profiles],
  );

  const refreshHistory = useCallback(async () => {
    setHistory(await backend.listDeploymentJobs(50));
  }, [backend]);

  const refreshJob = useCallback(async (jobID: string) => {
    try {
      setJob(await backend.getDeploymentJob(jobID));
      await refreshHistory();
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    }
  }, [backend, refreshHistory]);

  useEffect(() => {
    let active = true;
    Promise.all([backend.listDeploymentProjects(), backend.listSimulatorProfiles(), backend.listDeploymentJobs(50)])
      .then(async ([projectItems, profileItems, jobItems]) => {
        if (!active) return;
        setProjects(projectItems);
        setProfiles(profileItems);
        setHistory(jobItems);
        const latestJob = jobItems[0];
        if (latestJob) setJob(await backend.getDeploymentJob(latestJob.id));
        setSelectedProjectID((current) => current || projectItems[0]?.id || "");
        const supported = profileItems.find((profile) => profile.synthetic && profile.expectedCompatibility === "supported");
        setSelectedProfileID((current) => current || supported?.id || "");
      })
      .catch((cause: unknown) => { if (active) setError(errorMessage(cause)); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [backend]);

  useEffect(() => backend.onDeploymentProgress((event) => {
    if (job?.id === event.jobId) void refreshJob(event.jobId);
    else void refreshHistory();
  }), [backend, job?.id, refreshHistory, refreshJob]);

  const selectedProfile = supportedSyntheticProfiles.find((profile) => profile.id === selectedProfileID) ?? null;
  const active = job !== null && !isTerminal(job.state);

  async function startJob(mode: DeploymentMode) {
    if (!selectedProjectID || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const created = await backend.startDeploymentJob({
        projectId: selectedProjectID,
        ...(mode === DeploymentMode.DeploymentModeTest && selectedProfileID ? { targetProfileId: selectedProfileID } : {}),
        mode,
      });
      setJob(created);
      await refreshHistory();
      await refreshJob(created.id);
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setSubmitting(false);
    }
  }

  async function cancelJob() {
    if (!job || canceling) return;
    setCanceling(true);
    setError(null);
    try {
      setJob(await backend.cancelDeploymentJob(job.id));
      await refreshHistory();
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setCanceling(false);
    }
  }

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="eyebrow">CORE / DEPLOYMENT ENGINE</p>
          <h2 className="page-title mt-2">Развёртывание</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted">Пакет проходит preflight и синтетические проверки. Запись в автомобиль не выполняется.</p>
        </div>
        <Badge variant="warning" className="gap-2 px-3 py-2"><span className="size-1.5 rounded-full bg-warning" />PRODUCTION MODE · ЗАБЛОКИРОВАН</Badge>
      </header>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.12fr)_minmax(320px,.88fr)]">
        <Card className="overflow-hidden">
          <CardHeader>
            <div><CardTitle>Тестовая симуляция</CardTitle><p className="mt-1.5 text-xs text-muted">Локальный synthetic provider проверяет сохранённый CARVION package.</p></div>
            <span className="grid size-9 place-items-center rounded-xl border border-teal/15 bg-teal/8 text-teal"><PackageCheck size={17} aria-hidden="true" /></span>
          </CardHeader>
          <CardContent className="space-y-4 pt-5">
            {loading ? <Skeleton className="h-11" /> : projects.length ? (
              <label className="block space-y-2 text-[10px] font-semibold text-muted" htmlFor="deployment-project">
                Проект локализации
                <select id="deployment-project" aria-label="Проект локализации" value={selectedProjectID} onChange={(event) => setSelectedProjectID(event.target.value)} disabled={active || submitting} className="h-11 w-full rounded-xl border border-line bg-canvas/80 px-3 text-sm font-medium text-copy outline-none focus:border-accent/50 focus-visible:ring-2 focus-visible:ring-accent/50 disabled:opacity-50">
                  {projects.map((project) => <option key={project.id} value={project.id}>{project.name} · {project.locale}</option>)}
                </select>
              </label>
            ) : <EmptySelection text="Сначала создайте проект локализации и утвердите переводы." />}

            {loading ? <Skeleton className="h-11" /> : supportedSyntheticProfiles.length ? (
              <label className="block space-y-2 text-[10px] font-semibold text-muted" htmlFor="deployment-profile">
                Синтетический профиль
                <select id="deployment-profile" aria-label="Синтетический профиль" value={selectedProfileID} onChange={(event) => setSelectedProfileID(event.target.value)} disabled={active || submitting} className="h-11 w-full rounded-xl border border-line bg-canvas/80 px-3 text-sm font-medium text-copy outline-none focus:border-accent/50 focus-visible:ring-2 focus-visible:ring-accent/50 disabled:opacity-50">
                  {supportedSyntheticProfiles.map((profile) => <option key={profile.id} value={profile.id}>{profile.name} · TEST DATA</option>)}
                </select>
              </label>
            ) : <EmptySelection text="Нет синтетических профилей с точной supported записью." />}

            <div className="rounded-xl border border-teal/15 bg-teal/[0.035] p-3.5">
              <p className="flex items-center gap-2 text-xs font-semibold text-copy"><ShieldCheck size={15} className="text-teal" aria-hidden="true" />Тестовый provider не имеет возможности установки</p>
              <p className="mt-1.5 text-[10px] leading-4 text-muted">Проверка использует только локальный профиль и артефакт CARVION. Она не подтверждает работу на автомобиле.</p>
            </div>

            <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line pt-4">
              <p className="flex items-center gap-2 text-[10px] leading-4 text-faint"><Activity size={14} className="shrink-0 text-teal" aria-hidden="true" />Без автомобиля · без записи · история сохраняется в PostgreSQL</p>
              <div className="flex flex-wrap items-center gap-2">
                {active && job?.mode === DeploymentMode.DeploymentModeTest && <Button variant="danger" onClick={() => void cancelJob()} disabled={canceling} aria-label="Отменить тестовый прогон">{canceling ? <LoaderCircle size={14} className="animate-spin" /> : <Ban size={14} />}{canceling ? "Отменяем…" : "Отменить тестовый прогон"}</Button>}
                <Button onClick={() => void startJob(DeploymentMode.DeploymentModeTest)} disabled={loading || submitting || active || !selectedProjectID || !selectedProfile}>
                  {submitting ? <LoaderCircle size={15} className="animate-spin" aria-hidden="true" /> : <Play size={15} aria-hidden="true" />}
                  {submitting ? "Создаём задачу…" : "Запустить тестовый прогон"}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-warning/15 bg-warning/[0.025]">
          <CardHeader>
            <div><CardTitle>Production readiness</CardTitle><p className="mt-1.5 text-xs text-muted">Core сохраняет результат preflight и всегда запрещает Production запуск.</p></div>
            <ShieldAlert size={18} className="text-warning" aria-hidden="true" />
          </CardHeader>
          <CardContent className="space-y-4 pt-5">
            <div className="rounded-xl border border-warning/15 bg-warning/[0.045] p-3.5">
              <p className="text-xs font-semibold text-copy">Production Mode заблокирован в Go Core</p>
              <p className="mt-1.5 text-[10px] leading-4 text-muted">Аудит ниже сохраняется как заблокированная задача и не вызывает provider.</p>
            </div>
            <Button variant="secondary" className="w-full" onClick={() => void startJob(DeploymentMode.DeploymentModeProduction)} disabled={loading || submitting || active || !selectedProjectID}>
              {submitting ? <LoaderCircle size={14} className="animate-spin" aria-hidden="true" /> : <ShieldAlert size={14} aria-hidden="true" />}
              Проверить Production readiness
            </Button>
            <p className="text-[10px] leading-4 text-faint">Аудит проверит пакет и запишет причину блокировки. Это не команда на установку.</p>
          </CardContent>
        </Card>
      </div>

      {error && <div role="alert" className="rounded-xl border border-danger/20 bg-danger/5 px-4 py-3 text-xs leading-5 text-danger">{error}</div>}

      <Card>
        <CardHeader>
          <div><CardTitle>Предварительные проверки</CardTitle><p className="mt-1.5 text-xs text-muted">Результаты относятся к выбранной задаче. Целостность пакета не подтверждает его подлинность.</p></div>
          <Badge variant="neutral">{job ? labelForState(job.state, job.mode, job.preflight.checks) : "ОЖИДАНИЕ"}</Badge>
        </CardHeader>
        <CardContent className="space-y-4 pt-5">
          {job ? (
            <>
              <div role="status" aria-live="polite" className="rounded-xl border border-line bg-canvas/35 p-3.5">
                <p className="text-sm font-semibold text-copy">{labelForState(job.state, job.mode, job.preflight.checks)}</p>
                <p className="mt-1.5 text-xs leading-5 text-muted">{job.state === DeploymentState.DeploymentStateSimulated ? "Только симуляция; автомобиль не изменён" : job.errorMessage || job.events?.[job.events.length - 1]?.message || "Задача создана; ожидание события от Go Core."}</p>
                <p className="mt-2 break-all font-mono text-[9px] text-faint">Job {job.id} · SHA-256 {job.packageSha256}</p>
              </div>
              {job.preflight.checks?.length ? (
                <ul aria-label="Результаты предварительных проверок" className="grid gap-2 sm:grid-cols-2">
                  {job.preflight.checks.map((check) => <PreflightCheck key={check.id} check={check} />)}
                </ul>
              ) : <div className="rounded-xl border border-dashed border-line px-4 py-5 text-xs text-muted">После запуска Go Core сохранит статусы проверки пакета, совместимости, возможностей и восстановления.</div>}
              {!!job.resultVerification.message && <div className="rounded-xl border border-line bg-canvas/25 p-3 text-xs text-muted"><span className="font-semibold text-copy">Результат: </span>{job.resultVerification.message}</div>}
              {!!job.recovery.message && <div className="rounded-xl border border-line bg-canvas/25 p-3 text-xs text-muted"><span className="font-semibold text-copy">Восстановление: </span>{job.recovery.message}</div>}
            </>
          ) : <EmptySelection text="Выберите тестовый прогон или запустите аудит Production readiness." />}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div><CardTitle>События задачи</CardTitle><p className="mt-1.5 text-xs text-muted">Последовательность восстанавливается из постоянного аудита Core.</p></div>
          <Badge variant="neutral">{job ? `${job.events?.length ?? 0} СОБЫТИЙ` : "0 СОБЫТИЙ"}</Badge>
        </CardHeader>
        <CardContent className="pt-5">
          {job?.events?.length ? <ol aria-label="События развёртывания" className="divide-y divide-line">
            {job.events.map((event) => <li key={`${event.jobId}-${event.sequence}`} className="grid grid-cols-[24px_minmax(120px,.42fr)_minmax(0,1fr)_auto] items-start gap-3 py-3 first:pt-0 last:pb-0">
              <EventIcon state={event.state} />
              <code className="break-all pt-0.5 font-mono text-[10px] text-teal">{event.stage}</code>
              <p className="text-[11px] leading-5 text-muted">{event.message}</p>
              <time className="pt-0.5 font-mono text-[9px] tabular-nums text-faint" dateTime={event.at}>{formatTime(event.at)}</time>
            </li>)}
          </ol> : <EmptySelection text="После запуска здесь появятся события с сохранёнными порядковыми номерами." />}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div><CardTitle>История развёртываний</CardTitle><p className="mt-1.5 text-xs text-muted">Сохранённые тестовые и заблокированные Production задачи.</p></div>
          <Button variant="ghost" size="sm" onClick={() => void refreshHistory()} aria-label="Обновить историю"><RefreshCw size={14} aria-hidden="true" />Обновить историю</Button>
        </CardHeader>
        <CardContent className="pt-5">
          {history.length ? <ul aria-label="История развёртываний" className="divide-y divide-line">
            {history.map((item) => <li key={item.id}>
              <button type="button" onClick={() => void refreshJob(item.id)} className="grid w-full grid-cols-[minmax(0,1fr)_100px_auto] items-center gap-3 py-3 text-left first:pt-0 last:pb-0 hover:bg-panel-raised/40">
                <span className="min-w-0"><span className="block truncate text-xs font-semibold text-copy">{item.mode === DeploymentMode.DeploymentModeProduction ? "Production readiness" : "Тестовая симуляция"}</span><span className="mt-1 block truncate font-mono text-[9px] text-faint">{item.id} · {item.projectId}</span></span>
                <Badge variant={stateVariant(item.state)}>{item.state}</Badge>
                <time className="flex items-center gap-1.5 text-[9px] text-faint" dateTime={item.createdAt}><Clock3 size={11} aria-hidden="true" />{formatTime(item.createdAt)}</time>
              </button>
            </li>)}
          </ul> : <EmptySelection text="Сохранённых задач пока нет." />}
        </CardContent>
      </Card>
    </div>
  );
}

function PreflightCheck({ check }: Readonly<{ check: DeploymentCheck }>) {
  const label = checkLabels[check.id] ?? check.id;
  const variant = check.status === DeploymentCheckStatus.DeploymentCheckPassed ? "success" : check.status === DeploymentCheckStatus.DeploymentCheckFailed ? "danger" : check.status === DeploymentCheckStatus.DeploymentCheckUnknown ? "neutral" : "warning";
  return <li className="rounded-xl border border-line bg-canvas/25 p-3">
    <div className="flex items-center justify-between gap-2"><span className="text-[11px] font-semibold text-copy">{label}</span><Badge variant={variant}>{check.status}</Badge></div>
    <p className="mt-1.5 text-[10px] leading-4 text-muted">{check.message}</p>
  </li>;
}

function EmptySelection({ text }: Readonly<{ text: string }>) {
  return <div className="rounded-xl border border-dashed border-line px-4 py-6 text-center text-xs leading-5 text-muted">{text}</div>;
}

function EventIcon({ state }: Readonly<{ state: DeploymentState }>) {
  if (state === DeploymentState.DeploymentStateSimulated) return <CheckCircle2 size={16} className="text-accent" aria-hidden="true" />;
  if (state === DeploymentState.DeploymentStateBlocked || state === DeploymentState.DeploymentStateFailed) return <XCircle size={16} className="text-warning" aria-hidden="true" />;
  if (state === DeploymentState.DeploymentStateCancelled) return <Ban size={16} className="text-faint" aria-hidden="true" />;
  if (state === DeploymentState.DeploymentStatePreflighting || state === DeploymentState.DeploymentStateSimulating || state === DeploymentState.DeploymentStateVerifying) return <LoaderCircle size={16} className="animate-spin text-teal" aria-hidden="true" />;
  return <CheckCircle2 size={16} className="text-teal" aria-hidden="true" />;
}

function labelForState(state: DeploymentState, mode: DeploymentMode, checks: ReadonlyArray<DeploymentCheck> = []) {
  if (state === DeploymentState.DeploymentStateBlocked) return mode === DeploymentMode.DeploymentModeProduction ? "Production заблокирован" : "Требуется дополнительная проверка";
  if (state === DeploymentState.DeploymentStateFailed && checks.some((check) => check.id === "package-integrity" && check.status === DeploymentCheckStatus.DeploymentCheckFailed)) return "Проверка целостности пакета завершилась ошибкой";
  if (state === DeploymentState.DeploymentStateFailed) return "Тестовый прогон завершился с ошибкой";
  if (state === DeploymentState.DeploymentStateCancelled) return "Тестовый прогон отменён";
  if (state === DeploymentState.DeploymentStateSimulated) return "Симуляция завершена";
  if (state === DeploymentState.DeploymentStateSimulating || state === DeploymentState.DeploymentStateVerifying) return "Синтетический прогон выполняется";
  if (state === DeploymentState.DeploymentStatePreflighting || state === DeploymentState.DeploymentStatePreflightPassed) return "Выполняются предварительные проверки";
  return "Задача сохранена";
}

function isTerminal(state: DeploymentState) {
  return [DeploymentState.DeploymentStateBlocked, DeploymentState.DeploymentStateSimulated, DeploymentState.DeploymentStateFailed, DeploymentState.DeploymentStateCancelled].includes(state);
}

function stateVariant(state: DeploymentState) {
  if (state === DeploymentState.DeploymentStateSimulated) return "success" as const;
  if (state === DeploymentState.DeploymentStateFailed) return "danger" as const;
  if (state === DeploymentState.DeploymentStateBlocked) return "warning" as const;
  return "neutral" as const;
}

function formatTime(value: string) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? value : date.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

const checkLabels: Readonly<Record<string, string>> = {
  "package-integrity": "Целостность пакета",
  "target-profile": "Подтверждённая целевая платформа",
  compatibility: "Точное совпадение совместимости",
  "installation-capability": "Подтверждённый способ установки",
  "provider-capability": "Возможности провайдера",
  "result-verification": "Проверка результата",
  "recovery-capability": "Возможность восстановления",
  "bench-evidence": "Результаты стендовых испытаний",
  "production-policy": "Политика Production Mode",
};
