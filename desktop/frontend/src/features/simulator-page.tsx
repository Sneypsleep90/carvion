import { useEffect, useMemo, useRef, useState } from "react";
import {
  AlertTriangle,
  ArrowDownToLine,
  Check,
  CheckCircle2,
  Circle,
  Clock3,
  Cpu,
  FileCheck2,
  LoaderCircle,
  Play,
  Radio,
  RotateCcw,
  ShieldCheck,
  XCircle,
} from "lucide-react";
import { SimulatorRunStatus } from "../../bindings/github.com/Sneypsleep90/carvion/internal/core/domain/models";
import type { DesktopBackend, SimulatorProfile, SimulatorRunResult } from "../lib/backend";
import { downloadBase64, errorMessage } from "../lib/utils";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Skeleton } from "../components/ui/skeleton";

type SimulatorPageProps = Readonly<{ backend: DesktopBackend }>;

export function SimulatorPage({ backend }: SimulatorPageProps) {
  const [profiles, setProfiles] = useState<ReadonlyArray<SimulatorProfile>>([]);
  const [selectedID, setSelectedID] = useState("");
  const [result, setResult] = useState<SimulatorRunResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [canceling, setCanceling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const activeRequestID = useRef<string | null>(null);

  useEffect(() => {
    let active = true;
    backend.listSimulatorProfiles().then((items) => {
      if (!active) return;
      setProfiles(items);
      if (items.length) setSelectedID((current) => current || items[0]?.id || "");
    }).catch((cause: unknown) => {
      if (active) setError(errorMessage(cause));
    }).finally(() => {
      if (active) setLoading(false);
    });
    return () => { active = false; };
  }, [backend]);

  useEffect(() => () => {
    const requestID = activeRequestID.current;
    if (requestID) void backend.cancelSimulatorScenario(requestID);
  }, [backend]);

  const selectedProfile = useMemo(() => profiles.find((profile) => profile.id === selectedID) ?? null, [profiles, selectedID]);
  const runEvents = result?.events ?? [];

  async function runScenario() {
    if (!selectedID || running) return;
    setRunning(true);
    setCanceling(false);
    setError(null);
    setResult(null);
    const requestID = createRequestID();
    activeRequestID.current = requestID;
    try {
      setResult(await backend.runSimulatorScenario(selectedID, requestID));
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      activeRequestID.current = null;
      setCanceling(false);
      setRunning(false);
    }
  }

  async function cancelScenario() {
    const requestID = activeRequestID.current;
    if (!requestID || canceling) return;
    setCanceling(true);
    try {
      await backend.cancelSimulatorScenario(requestID);
    } catch (cause: unknown) {
      setError(errorMessage(cause));
      setCanceling(false);
    }
  }

  function changeProfile(profileID: string) {
    setSelectedID(profileID);
    setResult(null);
    setError(null);
  }

  const completed = result?.status === SimulatorRunStatus.SimulatorRunCompleted && result.packageVerified;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="eyebrow">LAB / SIMULATOR</p>
          <h2 className="page-title mt-2">Симулятор CARVION</h2>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-muted">
            Проведите полный тест локализации на виртуальном устройстве. Core использует тот же локальный JSON-RPC адаптер; автомобиль не подключается.
          </p>
        </div>
        <Badge variant="success" className="gap-2 px-3 py-2">
          <span className="size-1.5 rounded-full bg-accent" />
          SIMULATION · ТОЛЬКО ТЕСТОВЫЕ ДАННЫЕ
        </Badge>
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.12fr)_minmax(330px,.88fr)]">
        <Card className="overflow-hidden">
          <CardHeader>
            <div>
              <CardTitle>Испытательный профиль</CardTitle>
              <p className="mt-1.5 text-xs text-muted">Выбор из локального синтетического каталога.</p>
            </div>
            <span className="grid size-9 place-items-center rounded-xl border border-teal/15 bg-teal/8 text-teal"><Cpu size={17} aria-hidden="true" /></span>
          </CardHeader>
          <CardContent className="space-y-4 pt-5">
            {loading ? <Skeleton className="h-11" /> : profiles.length ? (
              <label className="block space-y-2 text-[10px] font-semibold text-muted" htmlFor="simulator-profile">
                Профиль симуляции
                <select
                  id="simulator-profile"
                  aria-label="Профиль симуляции"
                  value={selectedID}
                  onChange={(event) => changeProfile(event.target.value)}
                  disabled={running}
                  className="h-11 w-full rounded-xl border border-line bg-canvas/80 px-3 text-sm font-medium text-copy outline-none focus:border-accent/50 focus-visible:ring-2 focus-visible:ring-accent/50 disabled:opacity-50"
                >
                  {profiles.map((profile) => <option key={profile.id} value={profile.id}>{profile.name}</option>)}
                </select>
              </label>
            ) : (
              <div className="rounded-xl border border-dashed border-line px-4 py-6 text-center text-xs text-muted">Синтетические профили недоступны.</div>
            )}

            {selectedProfile && <ProfileSummary profile={selectedProfile} />}

            <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line pt-4">
              <p className="flex items-center gap-2 text-[10px] leading-4 text-faint">
                <ShieldCheck size={14} className="shrink-0 text-teal" aria-hidden="true" />
                Только CARVION TEST LAB · без настоящего автомобиля
              </p>
              <div className="flex items-center gap-2">
                {running && <Button variant="danger" onClick={() => void cancelScenario()} disabled={canceling} aria-label="Остановить сценарий">
                  {canceling ? <LoaderCircle size={14} className="animate-spin" aria-hidden="true" /> : null}
                  {canceling ? "Останавливаем…" : "Остановить"}
                </Button>}
                <Button onClick={() => void runScenario()} disabled={loading || running || !selectedProfile}>
                  {running ? <LoaderCircle size={15} className="animate-spin" aria-hidden="true" /> : <Play size={15} aria-hidden="true" />}
                  {running ? "Выполняем…" : result ? "Запустить снова" : "Запустить сценарий"}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="simulator-instrument overflow-hidden">
          <CardHeader>
            <div>
              <CardTitle>Результат прогона</CardTitle>
              <p className="mt-1.5 text-xs text-muted">Проверка совместимости и результат локализации.</p>
            </div>
            <Radio size={17} className="text-teal" aria-hidden="true" />
          </CardHeader>
          <CardContent className="pt-5">
            {running ? (
              <div role="status" aria-live="polite" className="rounded-xl border border-accent/15 bg-accent/[0.035] p-4">
                <p className="flex items-center gap-2 text-sm font-semibold text-copy"><LoaderCircle size={15} className="animate-spin text-accent" />Сценарий выполняется</p>
                <p className="mt-2 text-xs leading-5 text-muted">Go Core обращается к локальному адаптеру и готовит тестовый пакет.</p>
              </div>
            ) : result ? (
              <RunSummary result={result} completed={Boolean(completed)} onDownload={() => downloadBase64(result.package.filename, result.package.contentBase64, "application/zip")} />
            ) : (
              <div className="empty-state rounded-xl border border-dashed border-line px-4 py-8 text-center">
                <div className="mx-auto grid size-10 place-items-center rounded-xl border border-line bg-panel-raised text-faint"><RotateCcw size={17} /></div>
                <h3 className="mt-3 text-xs font-semibold text-copy">Стенд готов к прогону</h3>
                <p className="mt-1.5 text-[10px] leading-4 text-muted">Выберите профиль. Core обнаружит тестовое устройство и выполнит доступные проверки.</p>
              </div>
            )}
            {error && <div role="alert" className="mt-3 rounded-xl border border-danger/20 bg-danger/5 px-3 py-2.5 text-xs leading-5 text-danger">{error}</div>}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>Журнал сценария</CardTitle>
            <p className="mt-1.5 text-xs text-muted">События показывают только текущий прогон и исчезают при выборе другого профиля.</p>
          </div>
          <Badge variant="neutral">{result ? `${runEvents.length} СОБЫТИЙ` : "ОЖИДАНИЕ"}</Badge>
        </CardHeader>
        <CardContent className="pt-5">
          {runEvents.length ? (
            <ol aria-label="События симуляции" className="divide-y divide-line">
              {runEvents.map((event, index) => <li key={`${event.step}-${index}`} className="grid grid-cols-[24px_minmax(110px,.42fr)_minmax(0,1fr)_auto] items-start gap-3 py-3 first:pt-0 last:pb-0">
                <EventMarker status={event.status} />
                <code className="break-all pt-0.5 font-mono text-[10px] text-teal">{event.step}</code>
                <p className="text-[11px] leading-5 text-muted">{event.message}</p>
                <time className="pt-0.5 font-mono text-[9px] tabular-nums text-faint" dateTime={event.at}>{formatTime(event.at)}</time>
              </li>)}
            </ol>
          ) : (
            <div className="flex items-center gap-3 rounded-xl border border-dashed border-line px-4 py-5 text-xs text-muted">
              <Clock3 size={15} className="text-faint" aria-hidden="true" />
              После запуска здесь появится последовательность проверок, переданная Go Core.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function ProfileSummary({ profile }: Readonly<{ profile: SimulatorProfile }>) {
  const values = [
    { label: "Модель", value: profile.identity.Model },
    { label: "Платформа", value: profile.identity.PlatformID },
    { label: "Аппаратная ревизия", value: profile.hardware.Revision },
    { label: "Сборка ПО", value: profile.firmware.BuildID },
    { label: "Версия", value: profile.firmware.Version },
  ];
  return (
    <div className="overflow-hidden rounded-xl border border-line bg-canvas/45">
      <div className="flex items-center justify-between gap-3 border-b border-line px-3.5 py-3">
        <div className="min-w-0">
          <p className="truncate text-xs font-semibold text-copy">{profile.name}</p>
          <p className="mt-1 text-[10px] leading-4 text-muted">{profile.description}</p>
        </div>
        <Badge variant="info" className="shrink-0">SYNTHETIC</Badge>
      </div>
      <dl className="grid grid-cols-2 gap-x-4 gap-y-3 p-3.5 sm:grid-cols-3">
        {values.map((item) => <div key={item.label} className="min-w-0">
          <dt className="text-[9px] font-semibold uppercase tracking-[0.11em] text-faint">{item.label}</dt>
          <dd className="mt-1 truncate font-mono text-[10px] text-copy" title={item.value}>{item.value}</dd>
        </div>)}
      </dl>
    </div>
  );
}

function RunSummary({ result, completed, onDownload }: Readonly<{ result: SimulatorRunResult; completed: boolean; onDownload: () => void }>) {
  const label = result.status === SimulatorRunStatus.SimulatorRunCompleted ? "Сценарий завершён"
    : result.status === SimulatorRunStatus.SimulatorRunUnsupported ? "Аппаратная ревизия несовместима"
      : result.status === SimulatorRunStatus.SimulatorRunUnknown ? "Неизвестная сборка"
        : "Сценарий завершился с ошибкой";
  const icon = completed ? CheckCircle2 : result.status === SimulatorRunStatus.SimulatorRunUnsupported ? XCircle : result.status === SimulatorRunStatus.SimulatorRunUnknown ? AlertTriangle : XCircle;
  const Icon = icon;
  const tone = completed ? "text-accent" : result.status === SimulatorRunStatus.SimulatorRunUnknown ? "text-warning" : "text-danger";

  return (
    <div role="status" aria-live="polite" className="space-y-4">
      <div className="flex items-start gap-3 rounded-xl border border-line bg-canvas/45 p-3.5">
        <div className={`mt-0.5 grid size-8 shrink-0 place-items-center rounded-lg bg-panel-raised ${tone}`}><Icon size={17} /></div>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-copy">{label}</h3>
          <p className="mt-1 text-[10px] leading-4 text-muted">{result.compatibility.Reason || result.errorMessage || "Статус получен от локального адаптера."}</p>
        </div>
        <Badge variant={completed ? "success" : result.status === SimulatorRunStatus.SimulatorRunUnknown ? "warning" : "danger"}>{result.status.toUpperCase()}</Badge>
      </div>

      <dl className="grid grid-cols-2 gap-3">
        <ResultMetric label="Совместимость" value={result.compatibility.Status} />
        <ResultMetric label="Выбранный адаптер" value={result.adapterId || "Не выбран"} />
      </dl>

      {result.errorMessage && <p className="rounded-lg border border-danger/15 bg-danger/[0.04] px-3 py-2 text-[10px] leading-4 text-danger">{result.errorMessage}</p>}

      {completed && <div className="rounded-xl border border-accent/20 bg-accent/[0.035] p-3.5">
        <div className="flex items-center gap-2 text-xs font-semibold text-copy"><FileCheck2 size={15} className="text-accent" />Пакет проверен · OEM firmware: false</div>
        <p className="mt-1.5 break-all font-mono text-[9px] text-faint">SHA-256 · {result.package.sha256}</p>
        <p className="mt-1 text-[10px] leading-4 text-muted">Пакет CARVION · не OEM-прошивка. Установка не выполнялась.</p>
        <Button size="sm" className="mt-3 w-full" onClick={onDownload} disabled={!result.package.contentBase64}>
          <ArrowDownToLine size={14} aria-hidden="true" />Скачать языковой пакет
        </Button>
      </div>}

      <p className="border-t border-line pt-3 font-mono text-[9px] text-faint">RUN {result.runId}</p>
    </div>
  );
}

function ResultMetric({ label, value }: Readonly<{ label: string; value: string }>) {
  return <div className="min-w-0 rounded-lg border border-line bg-canvas/30 p-2.5">
    <dt className="text-[9px] font-semibold uppercase tracking-[0.12em] text-faint">{label}</dt>
    <dd className="mt-1 truncate font-mono text-[10px] text-copy" title={value}>{value || "—"}</dd>
  </div>;
}

function EventMarker({ status }: Readonly<{ status: string }>) {
  if (status === "completed" || status === "supported") return <Check size={14} className="mt-0.5 text-accent" aria-label="Завершено" />;
  if (status === "failed" || status === "unsupported") return <XCircle size={14} className="mt-0.5 text-danger" aria-label="Ошибка или отказ" />;
  if (status === "unknown") return <AlertTriangle size={14} className="mt-0.5 text-warning" aria-label="Неизвестно" />;
  return <Circle size={14} className="mt-0.5 text-faint" aria-label={status} />;
}

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "--:--:--";
  return new Intl.DateTimeFormat("ru-RU", { hour: "2-digit", minute: "2-digit", second: "2-digit", hourCycle: "h23" }).format(date);
}

function createRequestID(): string {
  return globalThis.crypto?.randomUUID?.() ?? `sim-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}
