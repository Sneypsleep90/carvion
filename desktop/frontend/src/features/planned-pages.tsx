import { ArrowUpRight, Clock3, Database, FileStack, Layers3, LockKeyhole, Sparkles } from "lucide-react";
import type { ApplicationState } from "../lib/backend";
import type { SectionId } from "../navigation";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent } from "../components/ui/card";

type LanguagePackagesPageProps = Readonly<{ onNavigate: (section: SectionId) => void }>;

export function LanguagePackagesPage({ onNavigate }: LanguagePackagesPageProps) {
  return (
    <PlannedPage
      eyebrow="ЛОКАЛИЗАЦИЯ / WORKFLOW"
      title="Языковые пакеты"
      description="Управление проектами переводов и сборка языковых пакетов планируются в Step 4 — Localization Engine."
      icon={FileStack}
      badge="STEP 4"
      points={[
        { icon: Layers3, title: "Проекты локализации", detail: "Статусы, языки и состав ресурсов" },
        { icon: LockKeyhole, title: "Проверяемые изменения", detail: "История review и подтверждение результата" },
        { icon: Sparkles, title: "Экспорт", detail: "Создание пакета после реализации workflow" },
      ]}
      actionLabel="Открыть инспекцию ресурсов"
      onAction={() => onNavigate("localization")}
    />
  );
}

export function OperationHistoryPage() {
  return (
    <div className="space-y-6">
      <div><p className="eyebrow">СИСТЕМА / АУДИТ</p><h2 className="page-title mt-2">История операций</h2><p className="mt-2 text-sm text-muted">Записи не создаются, пока в Core не появится хранилище истории.</p></div>
      <Card><CardContent className="empty-state py-16 text-center">
        <div className="mx-auto grid size-12 place-items-center rounded-2xl border border-line bg-panel-raised text-faint"><Clock3 size={21} /></div>
        <h3 className="mt-4 text-sm font-semibold text-copy">Операции не сохраняются в этой версии</h3>
        <p className="mx-auto mt-2 max-w-md text-xs leading-5 text-muted">История появится после определения модели аудита в следующих этапах. Здесь не показываются демонстрационные события.</p>
        <Badge variant="neutral" className="mt-4">EMPTY BY DESIGN</Badge>
      </CardContent></Card>
    </div>
  );
}

export function SettingsPage({ state }: Readonly<{ state: ApplicationState | null }>) {
  const rows = [
    { label: "Версия CARVION Studio", value: state?.version ?? "Загружается" },
    { label: "Go Core", value: state?.coreStatus ?? "Неизвестно" },
    { label: "Локальная PostgreSQL", value: state?.databaseStatus ?? "Неизвестно" },
    { label: "Обнаруженные адаптеры", value: state ? String(state.adapterCount) : "—" },
    { label: "Подключение к автомобилю", value: "Не поддерживается в Step 2" },
  ];
  return (
    <div className="space-y-6">
      <div><p className="eyebrow">СИСТЕМА / ОКРУЖЕНИЕ</p><h2 className="page-title mt-2">Настройки</h2><p className="mt-2 text-sm text-muted">Сведения об активном локальном Desktop runtime.</p></div>
      <div className="grid gap-4 xl:grid-cols-[1fr_0.72fr]">
        <Card><CardContent className="divide-y divide-line p-0">
          {rows.map((row) => <div key={row.label} className="flex items-center justify-between gap-4 px-5 py-4"><span className="text-xs text-muted">{row.label}</span><span className="text-right text-xs font-medium text-copy">{row.value}</span></div>)}
        </CardContent></Card>
        <Card className="border-accent/10 bg-accent/[0.025]"><CardContent className="p-5">
          <div className="grid size-9 place-items-center rounded-xl border border-accent/15 bg-accent/8 text-accent"><Database size={17} /></div>
          <h3 className="mt-4 text-sm font-semibold text-copy">Локальная среда разработки</h3>
          <p className="mt-2 text-xs leading-5 text-muted">В текущем этапе Studio использует PostgreSQL, настроенную через DATABASE_URL, и локальный каталог доверенных адаптеров. Настройки БД не передаются React.</p>
          <Badge variant={state?.databaseStatus === "connected" ? "success" : "warning"} className="mt-4">{state?.databaseStatus === "connected" ? "DB CONNECTED" : "DB UNAVAILABLE"}</Badge>
        </CardContent></Card>
      </div>
    </div>
  );
}

function PlannedPage({
  eyebrow,
  title,
  description,
  icon: Icon,
  badge,
  points,
  actionLabel,
  onAction,
}: Readonly<{
  eyebrow: string;
  title: string;
  description: string;
  icon: typeof FileStack;
  badge: string;
  points: ReadonlyArray<Readonly<{ icon: typeof Layers3; title: string; detail: string }>>;
  actionLabel: string;
  onAction: () => void;
}>) {
  return (
    <div className="space-y-6">
      <div><p className="eyebrow">{eyebrow}</p><h2 className="page-title mt-2">{title}</h2><p className="mt-2 max-w-2xl text-sm leading-6 text-muted">{description}</p></div>
      <Card className="overflow-hidden">
        <CardContent className="relative px-6 py-10 md:px-10 md:py-12">
          <div className="pointer-events-none absolute right-0 top-0 size-64 translate-x-1/4 -translate-y-1/3 rounded-full bg-teal/[0.035] blur-3xl" aria-hidden="true" />
          <div className="relative mx-auto max-w-3xl text-center">
            <Badge variant="neutral" className="mb-5">{badge} · ЗАПЛАНИРОВАНО</Badge>
            <div className="mx-auto grid size-14 place-items-center rounded-2xl border border-line bg-panel-raised text-teal"><Icon size={24} strokeWidth={1.6} /></div>
            <h3 className="mt-5 text-lg font-semibold tracking-tight text-copy">Эта область появится в следующем этапе</h3>
            <p className="mx-auto mt-2 max-w-xl text-xs leading-5 text-muted">CARVION Studio показывает планируемые возможности отдельно. Сохранение данных, операций и проектов здесь пока не имитируется.</p>
            <div className="mt-8 grid gap-3 text-left sm:grid-cols-3">
              {points.map((point) => {
                const PointIcon = point.icon;
                return <div key={point.title} className="rounded-xl border border-line bg-canvas/35 p-3.5"><PointIcon size={16} className="text-teal" /><p className="mt-3 text-xs font-semibold text-copy">{point.title}</p><p className="mt-1 text-[10px] leading-4 text-muted">{point.detail}</p></div>;
              })}
            </div>
            {onAction && <Button variant="secondary" className="mt-7" onClick={onAction}>{actionLabel}<ArrowUpRight size={14} /></Button>}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
