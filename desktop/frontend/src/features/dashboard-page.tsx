import { ArrowRight, BatteryCharging, Check, Database, Gauge, PlugZap, ShieldCheck, type LucideIcon } from "lucide-react";
import type { AdapterInfo, ApplicationState } from "../lib/backend";
import { formatAdapterCount } from "../lib/utils";
import type { SectionId } from "../navigation";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Skeleton } from "../components/ui/skeleton";

type DashboardPageProps = Readonly<{
  state: ApplicationState | null;
  stateLoading: boolean;
  stateError: string | null;
  adapters: ReadonlyArray<AdapterInfo> | null;
  adaptersLoading: boolean;
  adaptersError: string | null;
  onNavigate: (section: SectionId) => void;
}>;

export function DashboardPage({
  state,
  stateLoading,
  stateError,
  adapters,
  adaptersLoading,
  adaptersError,
  onNavigate,
}: DashboardPageProps) {
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="eyebrow">СТАНЦИЯ / ОБЗОР</p>
          <h2 className="page-title mt-2">Инженерная лаборатория</h2>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-muted">
            Локальная среда CARVION готова к работе с каталогом, адаптерами и синтетическими профилями.
          </p>
        </div>
        <Badge variant="info" className="gap-2 px-3 py-2">
          <ShieldCheck size={14} aria-hidden="true" />
          Без подключения к автомобилю
        </Badge>
      </div>

      <section className="dashboard-hero relative overflow-hidden rounded-2xl border border-line bg-panel px-6 py-6 md:px-8 md:py-7">
        <div className="hero-grid pointer-events-none absolute inset-0 opacity-60" aria-hidden="true" />
        <div className="relative flex flex-col justify-between gap-8 md:flex-row md:items-center">
          <div className="max-w-xl">
            <Badge variant="success" className="mb-4 border-accent/20 bg-accent/8 text-accent">
              <span className="size-1.5 rounded-full bg-accent" />
              СИМУЛЯЦИОННАЯ СРЕДА
            </Badge>
            <h3 className="text-[22px] font-semibold tracking-[-0.035em] text-copy sm:text-[26px]">
              Анализируйте данные<br className="hidden sm:block" /> в безопасном рабочем пространстве.
            </h3>
            <p className="mt-3 max-w-lg text-sm leading-6 text-muted">
              В этой сборке доступны каталог CARVION и тестовый Android-адаптер. Реальные автомобили и прошивки не используются.
            </p>
            <div className="mt-5 flex flex-wrap gap-2.5">
              <Button onClick={() => onNavigate("diagnostics")}>
                Открыть диагностику <ArrowRight size={15} aria-hidden="true" />
              </Button>
              <Button variant="secondary" onClick={() => onNavigate("adapters")}>
                Посмотреть адаптеры
              </Button>
            </div>
          </div>
          <div className="hero-instrument relative grid size-40 shrink-0 place-items-center self-center md:mr-5 md:size-48" aria-hidden="true">
            <div className="absolute inset-0 rounded-full border border-accent/10" />
            <div className="absolute inset-4 rounded-full border border-dashed border-teal/20" />
            <div className="absolute inset-8 rounded-full border border-line-strong bg-canvas/60" />
            <div className="absolute left-1/2 top-4 h-5 w-px -translate-x-1/2 bg-accent/60" />
            <Gauge size={42} strokeWidth={1.2} className="text-accent" />
            <span className="absolute bottom-7 text-[8px] font-bold tracking-[0.25em] text-faint">CORE READY</span>
            <span className="absolute right-1 top-1/2 size-1 rounded-full bg-accent shadow-[0_0_9px_2px_rgba(184,239,112,0.45)]" />
          </div>
        </div>
      </section>

      {stateError && <ErrorBanner message={stateError} />}
      {state?.message && <ErrorBanner message={state.message} />}
      {adaptersError && <ErrorBanner message={adaptersError} />}

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          title="Go Core"
          icon={Gauge}
          loading={stateLoading}
          value={state?.coreStatus === "ready" ? "Готов" : state?.coreStatus === "degraded" ? "Ограничен" : "Недоступен"}
          detail={state?.coreStatus === "ready" ? "Application Service отвечает" : "Проверьте локальную конфигурацию"}
          tone={state?.coreStatus === "ready" ? "accent" : "warning"}
        />
        <MetricCard
          title="Локальное хранилище"
          icon={Database}
          loading={stateLoading}
          value={state?.databaseStatus === "connected" ? "Подключено" : "Нет соединения"}
          detail="PostgreSQL · локальный режим разработки"
          tone={state?.databaseStatus === "connected" ? "teal" : "warning"}
        />
        <MetricCard
          title="Адаптеры"
          icon={PlugZap}
          loading={adaptersLoading}
          value={adapters ? formatAdapterCount(adapters.length) : "Загрузка"}
          detail={adapters?.length ? "Обнаружены локальным Plugin Manager" : "Нет зарегистрированных процессов"}
          tone={adapters?.length ? "accent" : "neutral"}
        />
        <MetricCard
          title="Сеанс автомобиля"
          icon={BatteryCharging}
          loading={false}
          value="Не подключён"
          detail="Подключение к автомобилю не реализовано"
          tone="neutral"
        />
      </div>

      <div className="grid gap-4 xl:grid-cols-[1.3fr_0.7fr]">
        <Card>
          <CardHeader>
            <div>
              <CardTitle>Состояние рабочей среды</CardTitle>
              <p className="mt-1.5 text-xs text-muted">Проверки выполняются локально внутри приложения.</p>
            </div>
            <Badge variant={state?.coreStatus === "ready" ? "success" : "warning"}>
              <span className="size-1.5 rounded-full bg-current" />
              {state?.coreStatus === "ready" ? "ОПЕРАЦИОННО" : "ТРЕБУЕТ ВНИМАНИЯ"}
            </Badge>
          </CardHeader>
          <CardContent className="pt-5">
            <div className="space-y-1">
              <CheckRow label="Wails Desktop host" value="Активен" state="done" />
              <CheckRow label="Прямые Go bindings" value="Активны" state="done" />
              <CheckRow label="PostgreSQL catalog" value={state?.databaseStatus === "connected" ? "Подключён" : "Нет соединения"} state={state?.databaseStatus === "connected" ? "done" : "warn"} />
              <CheckRow label="Доступ к автомобилю" value="Не используется" state="idle" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>Локальный набор</CardTitle>
              <p className="mt-1.5 text-xs text-muted">Только доверенные процессы из каталога.</p>
            </div>
            <span className="grid size-8 place-items-center rounded-lg border border-line bg-panel-raised text-teal">
              <PlugZap size={15} aria-hidden="true" />
            </span>
          </CardHeader>
          <CardContent className="pt-4">
            {adaptersLoading ? (
              <div className="space-y-3"><Skeleton className="h-12" /><Skeleton className="h-12" /></div>
            ) : adapters?.length ? (
              <div className="space-y-2">
                {adapters.slice(0, 2).map((adapter) => (
                  <div key={adapter.ID} className="flex items-center gap-3 rounded-xl border border-line bg-canvas/35 p-3">
                    <div className="grid size-8 shrink-0 place-items-center rounded-lg bg-teal/10 text-teal"><PlugZap size={15} /></div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-semibold text-copy">{adapter.Name}</p>
                      <p className="mt-1 truncate text-[10px] text-faint">{adapter.ID} · API {adapter.APIVersion}</p>
                    </div>
                    <Badge variant={adapter.ID === "adapter-android-demo" ? "info" : "neutral"}>
                      {adapter.ID === "adapter-android-demo" ? "TEST" : "LOCAL"}
                    </Badge>
                  </div>
                ))}
                <Button variant="ghost" size="sm" className="mt-1 w-full justify-between" onClick={() => onNavigate("adapters")}>
                  Открыть каталог адаптеров <ArrowRight size={14} />
                </Button>
              </div>
            ) : (
              <div className="rounded-xl border border-dashed border-line px-4 py-6 text-center">
                <PlugZap size={18} className="mx-auto text-faint" />
                <p className="mt-2 text-xs font-medium text-copy">Адаптеры не обнаружены</p>
                <p className="mt-1 text-[11px] text-muted">Соберите demo adapter и обновите каталог.</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function MetricCard({
  title,
  icon: Icon,
  value,
  detail,
  loading,
  tone,
}: Readonly<{
  title: string;
  icon: LucideIcon;
  value: string;
  detail: string;
  loading: boolean;
  tone: "accent" | "teal" | "warning" | "neutral";
}>) {
  const toneClass = {
    accent: "bg-accent/10 text-accent",
    teal: "bg-teal/10 text-teal",
    warning: "bg-warning/10 text-warning",
    neutral: "bg-panel-raised text-muted",
  }[tone];

  return (
    <Card className="metric-card">
      <CardContent className="p-4">
        <div className="flex items-center justify-between">
          <p className="text-[11px] font-medium text-muted">{title}</p>
          <span className={`grid size-8 place-items-center rounded-lg ${toneClass}`}><Icon size={15} aria-hidden="true" /></span>
        </div>
        {loading ? <Skeleton className="mt-4 h-6 w-28" /> : <p className="mt-4 truncate text-lg font-semibold tracking-tight text-copy">{value}</p>}
        <p className="mt-1 truncate text-[10px] text-faint">{detail}</p>
      </CardContent>
    </Card>
  );
}

function CheckRow({ label, value, state }: Readonly<{ label: string; value: string; state: "done" | "warn" | "idle" }>) {
  const iconClass = state === "done" ? "text-accent" : state === "warn" ? "text-warning" : "text-faint";
  const icon = state === "done" ? <Check size={13} /> : state === "warn" ? <span className="size-1.5 rounded-full bg-warning" /> : <span className="size-1.5 rounded-full border border-faint" />;
  return (
    <div className="flex items-center gap-3 border-b border-line/70 py-2.5 last:border-0">
      <span className={`grid size-5 place-items-center ${iconClass}`}>{icon}</span>
      <span className="flex-1 text-xs text-copy">{label}</span>
      <span className="text-[11px] text-muted">{value}</span>
    </div>
  );
}

function ErrorBanner({ message }: Readonly<{ message: string }>) {
  return (
    <div role="alert" className="rounded-xl border border-warning/20 bg-warning/5 px-4 py-3 text-xs leading-5 text-warning">
      <span className="font-semibold">Состояние требует внимания.</span> {message}
    </div>
  );
}
