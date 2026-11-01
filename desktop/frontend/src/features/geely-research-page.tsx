import { useEffect, useState, type ChangeEvent } from "react";
import { AlertCircle, ArrowRight, BookOpenText, CheckCircle2, Download, FileCode2, FileJson2, FileUp, Fingerprint, SearchCode, ShieldAlert } from "lucide-react";
import type { DesktopBackend, GeelyResearchAnalysis, GeelyResearchEvidence, GeelyResourceRequest } from "../lib/backend";
import { CompatibilityStatus, LocalizationFormat } from "../../bindings/github.com/Sneypsleep90/carvion/internal/core/domain/models";
import { downloadText, errorMessage } from "../lib/utils";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";

type GeelyResearchPageProps = Readonly<{ backend: DesktopBackend; onOpenLocalization: (projectID: string) => void }>;

const MAX_PROFILE_BYTES = 128 * 1024;
const MAX_RESOURCE_BYTES = 128 * 1024;

export function GeelyResearchPage({ backend, onOpenLocalization }: GeelyResearchPageProps) {
  const [evidence, setEvidence] = useState<ReadonlyArray<GeelyResearchEvidence>>([]);
  const [profileJSON, setProfileJSON] = useState("");
  const [profileFilename, setProfileFilename] = useState("");
  const [resource, setResource] = useState<GeelyResourceRequest | null>(null);
  const [analysis, setAnalysis] = useState<GeelyResearchAnalysis | null>(null);
  const [inspection, setInspection] = useState<Awaited<ReturnType<DesktopBackend["inspectGeelyResources"]>> | null>(null);
  const [createdProjectID, setCreatedProjectID] = useState("");
  const [projectName, setProjectName] = useState("Исследовательские ресурсы Geely");
  const [locale, setLocale] = useState("ru-RU");
  const [pending, setPending] = useState(false);
  const [loadingEvidence, setLoadingEvidence] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    void backend.listGeelyResearchEvidence().then((items) => {
      if (active) setEvidence(items);
    }).catch((cause: unknown) => {
      if (active) setError(errorMessage(cause));
    }).finally(() => {
      if (active) setLoadingEvidence(false);
    });
    return () => { active = false; };
  }, [backend]);

  async function chooseProfile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    try {
      const value = await readUTF8(file, MAX_PROFILE_BYTES);
      setProfileJSON(value);
      setProfileFilename(file.name);
      setAnalysis(null);
      setCreatedProjectID("");
      setError(null);
      setNotice(`Профиль «${file.name}» загружен локально. Его значения не подтверждены.`);
    } catch (cause: unknown) {
      setProfileJSON("");
      setProfileFilename("");
      setAnalysis(null);
      setCreatedProjectID("");
      setNotice(null);
      setError(errorMessage(cause));
    }
  }

  async function chooseResource(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    try {
      const content = await readUTF8(file, MAX_RESOURCE_BYTES);
      const format = file.name.toLowerCase().endsWith(".json")
        ? LocalizationFormat.LocalizationFormatJSON
        : LocalizationFormat.LocalizationFormatAndroidXML;
      setResource({ filename: file.name, format, content });
      setInspection(null);
      setCreatedProjectID("");
      setError(null);
      setNotice(`Ресурс «${file.name}» загружен. Содержимое останется локальным.`);
    } catch (cause: unknown) {
      setResource(null);
      setInspection(null);
      setCreatedProjectID("");
      setNotice(null);
      setError(errorMessage(cause));
    }
  }

  async function analyzeProfile() {
    if (!profileJSON) return;
    setPending(true);
    setError(null);
    setNotice(null);
    try {
      const result = await backend.analyzeGeelyProfile(profileJSON);
      setAnalysis(result);
      setNotice("Профиль проверен как импортированное наблюдение; соединения с автомобилем не было.");
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function inspectResource() {
    if (!resource) return;
    setPending(true);
    setError(null);
    setNotice(null);
    try {
      const result = await backend.inspectGeelyResources(resource);
      setInspection(result);
      setNotice(`Найдено переводимых ресурсов: ${result.Resources?.length ?? 0}.`);
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function importProject() {
    if (!analysis || !resource || !inspection) return;
    setPending(true);
    setError(null);
    setNotice(null);
    try {
      const result = await backend.importGeelyLocalizationProject({
        profileJson: profileJSON,
        name: projectName,
        locale,
        filename: resource.filename,
        format: resource.format,
        content: resource.content,
      });
      setCreatedProjectID(result.project.id);
      setNotice(`Создан локальный проект «${result.project.name}». Совместимость: ${compatibilityLabel(result.analysis.compatibility.Status)}.`);
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function exportReport() {
    if (!analysis) return;
    setPending(true);
    setError(null);
    try {
      const report = await backend.buildGeelyDiagnosticReport({
        profileJson: profileJSON,
        ...(resource ? { resource } : {}),
      });
      downloadText(report.filename, report.content, "application/json;charset=utf-8");
      setNotice("Диагностический отчёт экспортирован без исходного текста ресурсов и путей к файлам.");
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="eyebrow">RESEARCH / LOCAL FILES</p>
          <h2 className="page-title mt-2">Исследование Geely</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted">Локальный профиль и ресурсы мультимедиа для исследовательского анализа. Поддержка неизвестна, пока не появится проверенная точная запись.</p>
        </div>
        <Badge variant="warning"><SearchCode size={13} /> RESEARCH · БЕЗ ПОДКЛЮЧЕНИЯ</Badge>
      </div>

      <Card className="border-warning/20 bg-warning/[0.035]">
        <CardContent className="flex items-start gap-3 p-4">
          <ShieldAlert size={18} className="mt-0.5 shrink-0 text-warning" aria-hidden="true" />
          <div>
            <p className="text-xs font-semibold text-copy">Файловое исследование, не диагностика автомобиля</p>
            <p className="mt-1 text-xs leading-5 text-muted">Значения профиля берутся из выбранного JSON и остаются непроверенными. Для Xingyue L / Monjaro 2022–2024 в каталоге нет точных проверенных сочетаний платформы, региона, ревизии и сборки, поэтому результат остаётся unknown.</p>
          </div>
        </CardContent>
      </Card>

      {error && <div role="alert" className="rounded-xl border border-danger/20 bg-danger/5 px-4 py-3 text-xs text-danger"><AlertCircle className="mr-2 inline" size={14} />{error}</div>}
      {notice && <div role="status" className="rounded-xl border border-accent/20 bg-accent/5 px-4 py-3 text-xs text-accent">{notice}</div>}

      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <div><CardTitle>Профиль мультимедиа</CardTitle><p className="mt-1 text-[11px] text-muted">Версионированный JSON · carvion.geely-research/v1</p></div>
            <FileJson2 size={18} className="text-teal" aria-hidden="true" />
          </CardHeader>
          <CardContent className="space-y-4">
            <label className="block space-y-2 text-xs font-medium text-copy">
              <span>JSON-профиль</span>
              <Input aria-label="JSON профиль Geely" type="file" accept=".json,application/json" onChange={(event) => void chooseProfile(event)} />
            </label>
            {profileFilename && <p className="text-[11px] text-muted">Выбран файл: <span className="font-mono text-copy">{profileFilename}</span></p>}
            <Button onClick={() => void analyzeProfile()} disabled={pending || !profileJSON}>
              <Fingerprint size={15} /> Анализировать профиль
            </Button>
            {analysis && <ProfileAnalysis analysis={analysis} />}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div><CardTitle>Ресурсы интерфейса</CardTitle><p className="mt-1 text-[11px] text-muted">Отдельный Android strings.xml или carvion-json-v1</p></div>
            <FileCode2 size={18} className="text-teal" aria-hidden="true" />
          </CardHeader>
          <CardContent className="space-y-4">
            <label className="block space-y-2 text-xs font-medium text-copy">
              <span>XML/JSON-ресурс</span>
              <Input aria-label="XML или JSON ресурс" type="file" accept=".xml,.json,text/xml,application/json" onChange={(event) => void chooseResource(event)} />
            </label>
            {resource && <p className="text-[11px] text-muted">{resource.format} · <span className="font-mono text-copy">{resource.filename}</span></p>}
            <Button variant="secondary" onClick={() => void inspectResource()} disabled={pending || !resource}>
              <FileUp size={15} /> Анализировать ресурсы
            </Button>
            {inspection && <ResourceList resources={inspection.Resources ?? []} />}
          </CardContent>
        </Card>
      </div>

      {analysis && resource && inspection && (
        <Card>
          <CardHeader><div><CardTitle>Следующие действия</CardTitle><p className="mt-1 text-[11px] text-muted">Создание проекта сохраняет выбранный файл через Localization Engine, но не подразумевает установку.</p></div><Badge variant={statusVariant(analysis.compatibility.Status)}>{compatibilityLabel(analysis.compatibility.Status).toUpperCase()}</Badge></CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3 md:grid-cols-[1fr_180px]">
              <label className="space-y-1.5 text-[11px] text-muted">Название проекта<Input aria-label="Название проекта локализации" value={projectName} onChange={(event) => setProjectName(event.target.value)} /></label>
              <label className="space-y-1.5 text-[11px] text-muted">Язык перевода<Input aria-label="Целевой язык" value={locale} onChange={(event) => setLocale(event.target.value)} /></label>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button onClick={() => void importProject()} disabled={pending || !projectName.trim() || !locale.trim()}><FileUp size={15} /> Создать проект локализации</Button>
              <Button variant="secondary" onClick={() => void exportReport()} disabled={pending}><Download size={15} /> Экспортировать отчёт</Button>
              {createdProjectID && <Button variant="ghost" onClick={() => onOpenLocalization(createdProjectID)}><ArrowRight size={15} /> Открыть созданный проект</Button>}
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader><div><CardTitle>Публичные источники</CardTitle><p className="mt-1 text-[11px] text-muted">Источники подтверждают только сведения в указанной области и не доказывают совместимость.</p></div><BookOpenText size={17} className="text-teal" /></CardHeader>
        <CardContent>
          {loadingEvidence ? <p className="text-xs text-muted">Загрузка источников…</p> : evidence.length === 0 ? <p className="text-xs text-muted">Встроенные источники не доступны.</p> : (
            <div className="grid gap-3 lg:grid-cols-2">
              {evidence.map((source) => <article key={source.id} className="rounded-xl border border-line bg-canvas/50 p-4">
                <div className="flex items-start justify-between gap-3">
                  <a href={source.url} target="_blank" rel="noreferrer" className="text-xs font-semibold leading-5 text-copy underline decoration-line-strong underline-offset-4 hover:text-teal">{source.title}</a>
                  <Badge variant="neutral">{source.confidence}</Badge>
                </div>
                <p className="mt-2 text-[10px] text-muted">{source.publisher} · публикация {source.publishedAt || "не указана"} · получено {source.retrievedAt}</p>
                <p className="mt-2 text-[11px] leading-5 text-muted">{source.scope}</p>
              </article>)}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function ProfileAnalysis({ analysis }: Readonly<{ analysis: GeelyResearchAnalysis }>) {
  const profile = analysis.profile;
  const fields: ReadonlyArray<readonly [string, string]> = [
    ["Марка / модель", [profile.manufacturer, profile.model].filter(Boolean).join(" / ")],
    ["Год / рынок / комплектация", [profile.modelYear, profile.market, profile.trim].filter(Boolean).join(" / ")],
    ["Платформа", profile.platform ?? ""],
    ["ОС", [profile.osName, profile.osVersion].filter(Boolean).join(" ")],
    ["Аппаратная ревизия", [profile.hardwareIdentifier, profile.hardwareRevision].filter(Boolean).join(" / ")],
    ["Сборка ПО", [profile.firmwareVersion, profile.firmwareBuildId].filter(Boolean).join(" / ")],
  ];
  return (
    <div className="space-y-3 rounded-xl border border-line bg-canvas/60 p-4">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={statusVariant(analysis.compatibility.Status)}>{compatibilityLabel(analysis.compatibility.Status)}</Badge>
        <Badge variant="neutral">ИЗ ФАЙЛА · НЕ ПРОВЕРЕНО</Badge>
      </div>
      <p role="status" className="text-[11px] leading-5 text-muted">{analysis.compatibility.Reason}</p>
      <div className="grid gap-x-5 gap-y-2 sm:grid-cols-2">
        {fields.map(([label, value]) => <div key={label} className="min-w-0"><p className="text-[9px] font-semibold uppercase tracking-wide text-faint">{label}</p><p className="mt-0.5 truncate text-[11px] text-copy">{value || "unknown"}</p></div>)}
      </div>
      <p className="flex items-center gap-1.5 border-t border-line pt-3 text-[10px] text-muted"><CheckCircle2 size={13} className="text-teal" /> {analysis.detection.Reason}</p>
    </div>
  );
}

function ResourceList({ resources }: Readonly<{ resources: ReadonlyArray<{ ID: string; Text: string }> }>) {
  return (
    <div className="max-h-72 overflow-auto rounded-xl border border-line">
      <table className="w-full text-left text-[11px]">
        <thead className="sticky top-0 bg-panel-raised text-[9px] uppercase tracking-wide text-faint"><tr><th className="px-3 py-2">Ключ</th><th className="px-3 py-2">Исходный текст</th></tr></thead>
        <tbody>{resources.map((item) => <tr key={item.ID} className="border-t border-line"><td className="px-3 py-2 font-mono text-teal">{item.ID}</td><td className="px-3 py-2 text-copy">{item.Text}</td></tr>)}</tbody>
      </table>
      {resources.length === 0 && <p className="px-3 py-4 text-xs text-muted">Переводимых строк не найдено.</p>}
    </div>
  );
}

function compatibilityLabel(status: CompatibilityStatus): string {
  if (status === CompatibilityStatus.StatusSupported) return "Совместимо по проверенной записи";
  if (status === CompatibilityStatus.StatusUnsupported) return "Несовместимо по проверенной записи";
  return "Неизвестно · точной записи нет";
}

function statusVariant(status: CompatibilityStatus): "success" | "danger" | "warning" {
  if (status === CompatibilityStatus.StatusSupported) return "success";
  if (status === CompatibilityStatus.StatusUnsupported) return "danger";
  return "warning";
}

async function readUTF8(file: File, maximumBytes: number): Promise<string> {
  if (file.size > maximumBytes) {
    throw new Error(`Файл превышает допустимый размер ${maximumBytes} байт.`);
  }
  return new TextDecoder("utf-8", { fatal: true }).decode(await file.arrayBuffer());
}
