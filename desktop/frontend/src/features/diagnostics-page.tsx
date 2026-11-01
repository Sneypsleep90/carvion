import { useEffect, useState, type FormEvent } from "react";
import { AlertCircle, CheckCircle2, CircleHelp, FlaskConical, RotateCcw } from "lucide-react";
import type { AdapterInfo, CompatibilityRequest, CompatibilityResult, DesktopBackend } from "../lib/backend";
import { errorMessage } from "../lib/utils";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { AdapterPicker, adaptersWithCapability } from "../components/adapter-picker";

const sampleProfile: CompatibilityRequest = {
  Identity: { Manufacturer: "CARVION TEST LAB", Model: "Android Demo Unit", PlatformID: "android-demo-platform" },
  Hardware: { PlatformID: "android-demo-platform", Revision: "rev-a" },
  Firmware: { BuildID: "demo-build-a-001", Version: "1.0.0-demo" },
};

type FormValue = Readonly<{
  manufacturer: string;
  model: string;
  platformID: string;
  hardwareRevision: string;
  firmwareBuild: string;
  firmwareVersion: string;
}>;

type DiagnosticsPageProps = Readonly<{ backend: DesktopBackend; adapters: ReadonlyArray<AdapterInfo> }>;

export function DiagnosticsPage({ backend, adapters }: DiagnosticsPageProps) {
  const [selectedAdapterID, setSelectedAdapterID] = useState("");
  const [form, setForm] = useState<FormValue>({
    manufacturer: sampleProfile.Identity.Manufacturer,
    model: sampleProfile.Identity.Model,
    platformID: sampleProfile.Identity.PlatformID,
    hardwareRevision: sampleProfile.Hardware.Revision,
    firmwareBuild: sampleProfile.Firmware.BuildID,
    firmwareVersion: sampleProfile.Firmware.Version,
  });
  const [pending, setPending] = useState(false);
  const [result, setResult] = useState<CompatibilityResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const eligibleAdapters = adaptersWithCapability(adapters, "CheckCompatibility");

  useEffect(() => {
    if (!eligibleAdapters.some((adapter) => adapter.ID === selectedAdapterID)) {
      setSelectedAdapterID(eligibleAdapters[0]?.ID ?? "");
    }
  }, [adapters, selectedAdapterID]);

  function updateField(field: keyof FormValue, value: string) {
    setForm((current) => ({ ...current, [field]: value }));
    setResult(null);
    setError(null);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError(null);
    setResult(null);
    const request: CompatibilityRequest = {
      Identity: { Manufacturer: form.manufacturer, Model: form.model, PlatformID: form.platformID },
      Hardware: { PlatformID: form.platformID, Revision: form.hardwareRevision },
      Firmware: { BuildID: form.firmwareBuild, Version: form.firmwareVersion },
    };
    try {
      setResult(await backend.checkCompatibility(selectedAdapterID, request));
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  function resetSample() {
    setForm({
      manufacturer: sampleProfile.Identity.Manufacturer,
      model: sampleProfile.Identity.Model,
      platformID: sampleProfile.Identity.PlatformID,
      hardwareRevision: sampleProfile.Hardware.Revision,
      firmwareBuild: sampleProfile.Firmware.BuildID,
      firmwareVersion: sampleProfile.Firmware.Version,
    });
    setResult(null);
    setError(null);
  }

  return (
    <div className="space-y-6">
      <div>
        <p className="eyebrow">АНАЛИЗ / ПРОФИЛЬ</p>
        <h2 className="page-title mt-2">Проверка совместимости</h2>
        <p className="mt-2 max-w-2xl text-sm leading-6 text-muted">
          Данные отправляются напрямую из Wails binding в Go Core и тестовый адаптер. Это проверка синтетического профиля, не автомобиля.
        </p>
      </div>

      <div className="grid items-start gap-4 xl:grid-cols-[1fr_0.72fr]">
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="grid size-9 place-items-center rounded-xl border border-teal/15 bg-teal/8 text-teal"><FlaskConical size={17} /></div>
              <div><CardTitle>Параметры тестового профиля</CardTitle><p className="mt-1 text-[10px] text-muted">Совпадение проверяется по платформе, ревизии и точному build ID.</p></div>
            </div>
            <Badge variant="info">DEMO</Badge>
          </CardHeader>
          <CardContent>
            <form onSubmit={submit} className="space-y-5">
              <div className="grid gap-3 sm:grid-cols-2">
                <Field id="manufacturer" label="Производитель" value={form.manufacturer} onChange={(value) => updateField("manufacturer", value)} />
                <Field id="model" label="Модель" value={form.model} onChange={(value) => updateField("model", value)} />
                <Field id="platform-id" label="Platform ID" value={form.platformID} onChange={(value) => updateField("platformID", value)} />
                <Field id="hardware-revision" label="Аппаратная ревизия" value={form.hardwareRevision} onChange={(value) => updateField("hardwareRevision", value)} />
                <Field id="firmware-build" label="Firmware build ID" value={form.firmwareBuild} onChange={(value) => updateField("firmwareBuild", value)} />
                <Field id="firmware-version" label="Версия ПО" value={form.firmwareVersion} onChange={(value) => updateField("firmwareVersion", value)} />
              </div>
              <div className="flex flex-wrap items-end justify-between gap-3 border-t border-line pt-4">
                <AdapterPicker adapters={adapters} selectedAdapterID={selectedAdapterID} onChange={setSelectedAdapterID} capability="CheckCompatibility" label="Адаптер проверки" />
                <div className="flex gap-2">
                  <Button type="button" variant="ghost" onClick={resetSample}><RotateCcw size={13} /> Сбросить пример</Button>
                  <Button type="submit" disabled={pending || !selectedAdapterID}>{pending ? "Проверяем…" : "Проверить совместимость"}</Button>
                </div>
              </div>
            </form>
          </CardContent>
        </Card>

        <div className="space-y-4">
          <Card className="border-warning/15 bg-warning/[0.025]">
            <CardContent className="flex gap-3 p-4">
              <AlertCircle size={17} className="mt-0.5 shrink-0 text-warning" aria-hidden="true" />
              <div>
                <p className="text-xs font-semibold text-copy">Ограничение демо</p>
                <p className="mt-1 text-[11px] leading-5 text-muted">Используется выбранный адаптер. Неизвестные сборки и автомобили тестового адаптера возвращают `unknown`.</p>
              </div>
            </CardContent>
          </Card>
          <ResultCard result={result} error={error} pending={pending} />
        </div>
      </div>
    </div>
  );
}

function Field({ id, label, value, onChange }: Readonly<{ id: string; label: string; value: string; onChange: (value: string) => void }>) {
  return (
    <label htmlFor={id} className="block space-y-1.5">
      <span className="text-[10px] font-semibold text-muted">{label}</span>
      <Input id={id} value={value} onChange={(event) => onChange(event.target.value)} required spellCheck={false} />
    </label>
  );
}

function ResultCard({ result, error, pending }: Readonly<{ result: CompatibilityResult | null; error: string | null; pending: boolean }>) {
  const status = result?.Status;
  const variant = status === "supported" ? "success" : status === "unsupported" ? "danger" : "warning";
  const statusLabel = status === "supported" ? "Совместимо" : status === "unsupported" ? "Не поддерживается" : "Неизвестно";
  const Icon = status === "supported" ? CheckCircle2 : status === "unsupported" ? AlertCircle : CircleHelp;

  return (
    <Card>
      <CardHeader><CardTitle>Результат анализа</CardTitle>{result && <Badge variant={variant}>{statusLabel}</Badge>}</CardHeader>
      <CardContent>
        {pending ? (
          <div role="status" className="rounded-xl border border-line bg-canvas/40 p-5 text-center">
            <span className="loading-orbit mx-auto grid size-10 place-items-center rounded-full border border-accent/20 text-accent"><FlaskConical size={17} /></span>
            <p className="mt-3 text-xs font-semibold text-copy">Адаптер проверяет профиль</p>
            <p className="mt-1 text-[10px] text-muted">Ответ ожидается от локального JSON-RPC процесса.</p>
          </div>
        ) : result ? (
          <div role="status" className={`rounded-xl border p-4 ${status === "supported" ? "border-accent/20 bg-accent/[0.04]" : status === "unsupported" ? "border-danger/20 bg-danger/[0.04]" : "border-warning/20 bg-warning/[0.04]"}`}>
            <div className="flex items-center gap-2.5">
              <Icon size={19} className={status === "supported" ? "text-accent" : status === "unsupported" ? "text-danger" : "text-warning"} aria-hidden="true" />
              <p className="text-sm font-semibold text-copy">{statusLabel}</p>
            </div>
            <p className="mt-3 text-xs leading-5 text-muted">{result.Reason}</p>
            <p className="mt-3 border-t border-line/80 pt-3 text-[10px] text-faint">Результат относится только к синтетическому профилю CARVION TEST LAB.</p>
          </div>
        ) : error ? (
          <div role="alert" className="rounded-xl border border-danger/20 bg-danger/[0.04] p-4 text-xs leading-5 text-danger">{error}</div>
        ) : (
          <div className="rounded-xl border border-dashed border-line px-4 py-8 text-center">
            <CircleHelp size={20} className="mx-auto text-faint" />
            <p className="mt-2 text-xs font-medium text-copy">Ожидает профиль</p>
            <p className="mt-1 text-[10px] text-muted">Запустите проверку, чтобы получить `supported`, `unsupported` или `unknown`.</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
