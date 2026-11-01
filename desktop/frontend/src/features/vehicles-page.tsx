import { useEffect, useState } from "react";
import { CarFront, Database, Search, ShieldCheck } from "lucide-react";
import type { DesktopBackend, InfotainmentPlatform, Manufacturer, VehicleModel } from "../lib/backend";
import { errorMessage } from "../lib/utils";
import { Badge } from "../components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Skeleton } from "../components/ui/skeleton";

type VehiclesPageProps = Readonly<{ backend: DesktopBackend }>;

export function VehiclesPage({ backend }: VehiclesPageProps) {
  const [manufacturers, setManufacturers] = useState<ReadonlyArray<Manufacturer>>([]);
  const [models, setModels] = useState<ReadonlyArray<VehicleModel>>([]);
  const [platforms, setPlatforms] = useState<ReadonlyArray<InfotainmentPlatform>>([]);
  const [selectedManufacturer, setSelectedManufacturer] = useState("");
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    backend.listManufacturers().then((items) => {
      if (!active) return;
      setManufacturers(items);
      setSelectedManufacturer(items[0]?.ID ?? "");
    }).catch((cause: unknown) => {
      if (active) setError(errorMessage(cause));
    }).finally(() => {
      if (active) setLoading(false);
    });
    return () => { active = false; };
  }, [backend]);

  useEffect(() => {
    if (!selectedManufacturer) {
      setModels([]);
      setPlatforms([]);
      return;
    }
    let active = true;
    setDetailLoading(true);
    Promise.all([
      backend.listVehicleModels(selectedManufacturer),
      backend.listInfotainmentPlatforms(selectedManufacturer),
    ]).then(([modelItems, platformItems]) => {
      if (!active) return;
      setModels(modelItems);
      setPlatforms(platformItems);
    }).catch((cause: unknown) => {
      if (active) setError(errorMessage(cause));
    }).finally(() => {
      if (active) setDetailLoading(false);
    });
    return () => { active = false; };
  }, [backend, selectedManufacturer]);

  return (
    <div className="space-y-6">
      <div>
        <p className="eyebrow">КАТАЛОГ / ЛОКАЛЬНЫЙ</p>
        <h2 className="page-title mt-2">Автомобили</h2>
        <p className="mt-2 text-sm text-muted">Показываются записи из PostgreSQL catalog; соединение с автомобилем не выполняется.</p>
      </div>

      <Card className="border-accent/10 bg-accent/[0.025]">
        <CardContent className="flex items-start gap-3 p-4">
          <ShieldCheck size={17} className="mt-0.5 text-accent" aria-hidden="true" />
          <p className="text-xs leading-5 text-muted">
            Этот каталог является справочником локального окружения. Наличие записи не подтверждает совместимость или физическое подключение.
          </p>
        </CardContent>
      </Card>

      {error && <div role="alert" className="rounded-xl border border-danger/20 bg-danger/5 px-4 py-3 text-xs text-danger">{error}</div>}

      {loading ? (
        <div className="grid gap-4 lg:grid-cols-2"><Skeleton className="h-64" /><Skeleton className="h-64" /></div>
      ) : manufacturers.length === 0 ? (
        <Card>
          <CardContent className="empty-state py-16 text-center">
            <div className="mx-auto grid size-12 place-items-center rounded-2xl border border-line bg-panel-raised text-faint"><CarFront size={22} /></div>
            <h3 className="mt-4 text-sm font-semibold text-copy">Локальный каталог пока пуст</h3>
            <p className="mx-auto mt-2 max-w-md text-xs leading-5 text-muted">
              В PostgreSQL нет записей производителей. CARVION Studio не подставляет фиктивные автомобили и не заявляет поддержку платформ.
            </p>
            <Badge variant="neutral" className="mt-4"><Database size={12} /> PostgreSQL catalog</Badge>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 xl:grid-cols-[0.78fr_1.22fr]">
          <Card>
            <CardHeader><CardTitle>Производители</CardTitle><Badge variant="neutral">{manufacturers.length}</Badge></CardHeader>
            <CardContent className="space-y-2">
              {manufacturers.map((manufacturer) => (
                <button
                  key={manufacturer.ID}
                  type="button"
                  onClick={() => setSelectedManufacturer(manufacturer.ID)}
                  className={`flex w-full items-center gap-3 rounded-xl border p-3 text-left transition ${selectedManufacturer === manufacturer.ID ? "border-accent/25 bg-accent/5" : "border-line bg-canvas/30 hover:bg-panel-raised"}`}
                >
                  <div className="grid size-8 place-items-center rounded-lg bg-panel-raised text-muted"><CarFront size={15} /></div>
                  <span className="flex-1 text-xs font-semibold text-copy">{manufacturer.Name}</span>
                  {selectedManufacturer === manufacturer.ID && <span className="size-1.5 rounded-full bg-accent" />}
                </button>
              ))}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <div><CardTitle>Модели и мультимедийные платформы</CardTitle><p className="mt-1.5 text-xs text-muted">{manufacturers.find((item) => item.ID === selectedManufacturer)?.Name}</p></div>
              <Search size={15} className="text-faint" />
            </CardHeader>
            <CardContent>
              {detailLoading ? <Skeleton className="h-32" /> : (
                <div className="grid gap-4 md:grid-cols-2">
                  <CatalogColumn title="Модели" empty="Модели не зарегистрированы" values={models.map((item) => `${item.Name} · ${item.ID}`)} />
                  <CatalogColumn title="Платформы" empty="Платформы не зарегистрированы" values={platforms.map((item) => `${item.Name} · ${item.ID}`)} />
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}

function CatalogColumn({ title, empty, values }: Readonly<{ title: string; empty: string; values: ReadonlyArray<string> }>) {
  return (
    <div>
      <p className="mb-2 text-[9px] font-bold uppercase tracking-[0.16em] text-faint">{title}</p>
      {values.length ? <ul className="space-y-2">{values.map((value) => <li key={value} className="rounded-lg border border-line bg-canvas/30 px-3 py-2 text-[11px] text-copy">{value}</li>)}</ul> : <p className="rounded-lg border border-dashed border-line px-3 py-4 text-center text-[10px] text-muted">{empty}</p>}
    </div>
  );
}
