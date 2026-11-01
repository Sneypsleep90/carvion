import { BadgeCheck, Cpu, ExternalLink, FileCode2, PlugZap, ShieldAlert } from "lucide-react";
import type { AdapterInfo, ApplicationState } from "../lib/backend";
import { AdapterStatus } from "../lib/backend";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Skeleton } from "../components/ui/skeleton";

type AdaptersPageProps = Readonly<{
  adapters: ReadonlyArray<AdapterInfo> | null;
  loading: boolean;
  error: string | null;
  state: ApplicationState | null;
  onRefresh: () => void;
}>;

export function AdaptersPage({ adapters, loading, error, state, onRefresh }: AdaptersPageProps) {
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="eyebrow">СИСТЕМА / ПРОЦЕССЫ</p>
          <h2 className="page-title mt-2">Адаптеры</h2>
          <p className="mt-2 text-sm text-muted">Список сформирован локальным Plugin Manager из доверенного каталога.</p>
        </div>
        <Button variant="secondary" onClick={onRefresh} disabled={loading}>
          <PlugZap size={14} aria-hidden="true" />
          Обновить список
        </Button>
      </div>

      <Card className="border-teal/15 bg-teal/[0.035]">
        <CardContent className="flex items-start gap-3 p-4">
          <ShieldAlert size={18} className="mt-0.5 shrink-0 text-teal" aria-hidden="true" />
          <div>
            <p className="text-xs font-semibold text-copy">Процессы остаются локальными</p>
            <p className="mt-1 text-xs leading-5 text-muted">
              Адаптеры запускаются Go Core через JSON-RPC. Каталог не загружает код из сети; отдельный процесс не является OS sandbox.
            </p>
          </div>
          <Badge variant="info" className="ml-auto shrink-0">{state?.adapterCount ?? "—"} обнаружено</Badge>
        </CardContent>
      </Card>

      {error && <div role="alert" className="rounded-xl border border-danger/20 bg-danger/5 px-4 py-3 text-xs text-danger">{error}</div>}

      {loading ? (
        <div className="grid gap-4 lg:grid-cols-2"><Skeleton className="h-52" /><Skeleton className="h-52" /></div>
      ) : adapters?.length ? (
        <div className="grid gap-4 lg:grid-cols-2">
          {adapters.map((adapter) => <AdapterCard key={adapter.ID} adapter={adapter} />)}
        </div>
      ) : (
        <Card>
          <CardContent className="empty-state py-14 text-center">
            <div className="mx-auto grid size-12 place-items-center rounded-2xl border border-line bg-panel-raised text-faint"><PlugZap size={21} /></div>
            <h3 className="mt-4 text-sm font-semibold text-copy">Локальные адаптеры не найдены</h3>
            <p className="mx-auto mt-2 max-w-md text-xs leading-5 text-muted">
              Проверьте ADAPTERS_DIR и убедитесь, что demo adapter собран. Пустой список не означает поддержку какой-либо платформы.
            </p>
            <Button variant="secondary" size="sm" className="mt-4" onClick={onRefresh}>Повторить поиск</Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function AdapterCard({ adapter }: Readonly<{ adapter: AdapterInfo }>) {
  const isDemo = adapter.ID === "adapter-android-demo";
  const isAvailable = adapter.Status === AdapterStatus.AdapterStatusAvailable;
  const capabilities = [
    { label: "Определение", enabled: adapter.Capabilities.Detect },
    { label: "Совместимость", enabled: adapter.Capabilities.CheckCompatibility },
    { label: "Ресурсы", enabled: adapter.Capabilities.InspectResources },
  ];

  return (
    <Card className="overflow-hidden">
      <CardHeader>
        <div className="flex min-w-0 items-center gap-3">
          <div className="grid size-11 shrink-0 place-items-center rounded-xl border border-teal/15 bg-teal/8 text-teal"><Cpu size={21} /></div>
          <div className="min-w-0">
            <CardTitle className="truncate text-sm">{adapter.Name}</CardTitle>
            <p className="mt-1 truncate font-mono text-[10px] text-faint">{adapter.ID}</p>
          </div>
        </div>
        <Badge variant={isAvailable ? (isDemo ? "info" : "success") : "danger"}>{isAvailable ? (isDemo ? "SYNTHETIC" : "LOCAL · ДОСТУПЕН") : "ПОСЛЕДНИЙ ВЫЗОВ: ОШИБКА"}</Badge>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-2 text-[11px] text-muted">
          <span>Адаптер {adapter.Version}</span><span className="text-line-strong">/</span><span>SDK {adapter.SDKVersion}</span><span className="text-line-strong">/</span><span>API {adapter.APIVersion}</span>
        </div>
        {isDemo && (
          <div className="mt-3 flex items-center gap-2 rounded-lg border border-teal/15 bg-teal/[0.04] px-3 py-2 text-[11px] text-teal">
            <BadgeCheck size={14} aria-hidden="true" />
            <span>Только синтетические данные</span>
          </div>
        )}
        <div className="mt-4 border-t border-line pt-3">
          <p className="text-[9px] font-bold uppercase tracking-[0.16em] text-faint">Возможности</p>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {capabilities.map((capability) => (
              <Badge key={capability.label} variant={capability.enabled ? "success" : "neutral"}>
                {capability.enabled && <span className="size-1 rounded-full bg-current" />}
                {capability.label}
              </Badge>
            ))}
          </div>
        </div>
        <div className="mt-4 grid gap-3 border-t border-line pt-3 sm:grid-cols-2">
          <MetadataList title="Входные данные" values={adapter.InputTypes ?? []} emptyLabel="Не объявлены" />
          <MetadataList title="Форматы ресурсов" values={adapter.ResourceFormats ?? []} emptyLabel="Не объявлены" />
        </div>
        <div className="mt-4 flex items-center justify-between border-t border-line pt-3">
          <span className="inline-flex items-center gap-1.5 text-[10px] text-muted"><FileCode2 size={12} /> JSON-RPC 2.0 / stdin · stdout</span>
          <Button variant="ghost" size="sm" className="h-7 px-2 text-[10px]" disabled aria-label="Администрирование адаптера не реализовано">
            <ExternalLink size={12} />
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function MetadataList({ title, values, emptyLabel }: Readonly<{ title: string; values: ReadonlyArray<string>; emptyLabel: string }>) {
  return (
    <div className="min-w-0">
      <p className="text-[9px] font-bold uppercase tracking-[0.16em] text-faint">{title}</p>
      <div className="mt-2 flex flex-wrap gap-1.5">
        {values.length ? values.map((value) => <Badge key={value} variant="neutral" className="max-w-full truncate font-mono text-[9px]">{value}</Badge>) : <span className="text-[10px] text-muted">{emptyLabel}</span>}
      </div>
    </div>
  );
}
