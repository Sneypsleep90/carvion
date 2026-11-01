import { useEffect, useMemo, useRef, useState, type ChangeEvent } from "react";
import { AlertCircle, Check, CheckCircle2, Download, FileCode2, FileJson2, FileUp, Languages, LoaderCircle, Package, Plus, Search, ShieldCheck } from "lucide-react";
import type { DesktopBackend, LocalizationArtifact, LocalizationProject, LocalizationProjectSummary, LocalizationTranslation, LocalizationValidationReport } from "../lib/backend";
import { LocalizationFormat } from "../../bindings/github.com/Sneypsleep90/carvion/internal/core/domain/models";
import { downloadBase64, errorMessage } from "../lib/utils";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";

const sampleXML = `<?xml version="1.0" encoding="utf-8"?>
<resources>
  <string name="welcome_title">Synthetic media</string>
  <string name="welcome_message">Hello, %1$s</string>
  <string-array name="audio_modes"><item>Standard</item><item>Night</item></string-array>
  <plurals name="track_count"><item quantity="one">%d track</item><item quantity="other">%d tracks</item></plurals>
  <string name="internal_route" translatable="false">synthetic-route</string>
</resources>`;

type LocalizationPageProps = Readonly<{ backend: DesktopBackend; initialProjectID: string | null; onInitialProjectLoaded: () => void }>;

export function LocalizationPage({ backend, initialProjectID, onInitialProjectLoaded }: LocalizationPageProps) {
  const [projects, setProjects] = useState<ReadonlyArray<LocalizationProjectSummary>>([]);
  const [project, setProject] = useState<LocalizationProject | null>(null);
  const [name, setName] = useState("Новый проект локализации");
  const [locale, setLocale] = useState("ru-RU");
  const [format, setFormat] = useState<LocalizationFormat>(LocalizationFormat.LocalizationFormatAndroidXML);
  const [filename, setFilename] = useState("strings.xml");
  const [content, setContent] = useState("");
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [search, setSearch] = useState("");
  const [report, setReport] = useState<LocalizationValidationReport | null>(null);
  const [pending, setPending] = useState(false);
  const [loadingProjects, setLoadingProjects] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const sourceFile = useRef<HTMLInputElement>(null);
  const csvFile = useRef<HTMLInputElement>(null);
  const replaceSourceMode = useRef(false);

  useEffect(() => {
    void refreshProjects();
    // Project listing is a one-time load; updates are refreshed after mutations.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!initialProjectID) return;
    void backend.getLocalizationProject(initialProjectID).then((loaded) => {
      setLoadedProject(loaded);
      setError(null);
    }).catch((cause: unknown) => {
      setError(errorMessage(cause));
    }).finally(onInitialProjectLoaded);
  }, [backend, initialProjectID, onInitialProjectLoaded]);

  const translations = useMemo(() => new Map((project?.translations ?? []).map((item) => [item.resourceKey, item])), [project]);
  const hasUnsavedChanges = useMemo(() => (project?.resources ?? []).some((resource) =>
    (drafts[resource.key] ?? translations.get(resource.key)?.targetText ?? "") !== (translations.get(resource.key)?.targetText ?? ""),
  ), [project, drafts, translations]);
  const visibleResources = useMemo(() => (project?.resources ?? []).filter((resource) => {
    const query = search.trim().toLocaleLowerCase();
    return !query || resource.key.toLocaleLowerCase().includes(query) || resource.sourceText.toLocaleLowerCase().includes(query) || (drafts[resource.key] ?? translations.get(resource.key)?.targetText ?? "").toLocaleLowerCase().includes(query);
  }), [project, search, drafts, translations]);

  async function refreshProjects() {
    setLoadingProjects(true);
    try {
      setProjects(await backend.listLocalizationProjects());
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setLoadingProjects(false);
    }
  }

  function setLoadedProject(value: LocalizationProject) {
    setProject(value);
    setLocale(value.locale);
    setName(value.name);
    setFilename(value.filename);
    setFormat(value.format);
    setDrafts(Object.fromEntries((value.translations ?? []).map((translation) => [translation.resourceKey, translation.targetText])));
    setReport(null);
    setNotice(null);
  }

  async function importProject() {
    if (!content.trim()) return;
    setPending(true);
    setError(null);
    setNotice(null);
    try {
      const created = await backend.importLocalizationProject(name, locale, format, filename, content);
      setLoadedProject(created);
      await refreshProjects();
      setNotice(`Импортировано ресурсов: ${created.resources?.length ?? 0}. Исходный файл сохранён без изменений.`);
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function chooseSource(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    try {
      const detected = file.name.toLowerCase().endsWith(".json") ? LocalizationFormat.LocalizationFormatJSON : LocalizationFormat.LocalizationFormatAndroidXML;
      const fileContent = await readUTF8(file);
      if (replaceSourceMode.current && project) {
        replaceSourceMode.current = false;
        if (detected !== project.format) throw new Error("Формат исходного файла должен совпадать с форматом проекта.");
        setLoadedProject(await backend.replaceLocalizationSource(project.id, file.name, fileContent));
        await refreshProjects();
        setNotice("Источник обновлён. Утверждённые переводы изменившихся строк требуют повторной проверки.");
        return;
      }
      replaceSourceMode.current = false;
      setContent(fileContent);
      setFilename(file.name);
      setFormat(detected);
      setName(file.name.replace(/\.[^.]+$/, ""));
      setProject(null);
      setReport(null);
      setError(null);
      setNotice(`Выбран файл ${file.name}`);
    } catch (cause: unknown) {
      replaceSourceMode.current = false;
      setError(errorMessage(cause));
    }
  }

  async function chooseCSV(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file || !project) return;
    setPending(true);
    setError(null);
    try {
      setLoadedProject(await backend.importLocalizationCSV(project.id, await readUTF8(file)));
      setNotice(`Переводы импортированы из ${file.name} как черновики.`);
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function openProject(projectID: string) {
    if (!projectID) {
      setProject(null);
      setReport(null);
      return;
    }
    setPending(true);
    setError(null);
    try {
      setLoadedProject(await backend.getLocalizationProject(projectID));
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function saveTranslation(resourceKey: string) {
    if (!project) return;
    setPending(true);
    setError(null);
    try {
      const saved = await backend.updateLocalizationTranslation(project.id, resourceKey, drafts[resourceKey] ?? "");
      setProject((current) => current ? updateTranslation(current, saved) : current);
      setReport(null);
      setNotice(`Черновик «${resourceKey}» сохранён.`);
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function approveTranslation(resourceKey: string) {
    if (!project) return;
    setPending(true);
    setError(null);
    try {
      const approved = await backend.approveLocalizationTranslation(project.id, resourceKey);
      setProject((current) => current ? updateTranslation(current, approved) : current);
      setReport(null);
      setNotice(`Перевод «${resourceKey}» утверждён для текущей исходной строки.`);
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function validate() {
    if (!project) return;
    setPending(true);
    setError(null);
    try {
      setReport(await backend.validateLocalizationProject(project.id));
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function exportCSV() {
    if (!project) return;
    await runArtifact(() => backend.exportLocalizationCSV(project.id));
  }

  async function exportResources() {
    if (!project) return;
    await runArtifact(() => backend.exportLocalizationResources(project.id));
  }

  async function buildPackage() {
    if (!project) return;
    setPending(true);
    setError(null);
    try {
      const item = await backend.buildLocalizationPackage(project.id);
      downloadBase64(item.filename, item.contentBase64, "application/zip");
      setNotice(`Пакет ${item.filename} собран. SHA-256: ${item.sha256}`);
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function runArtifact(action: () => Promise<LocalizationArtifact>) {
    setPending(true);
    setError(null);
    try {
      const artifact = await action();
      downloadBase64(artifact.filename, artifact.contentBase64, artifact.filename.endsWith(".csv") ? "text/csv;charset=utf-8" : "application/octet-stream");
      setNotice(`Файл ${artifact.filename} подготовлен.`);
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
          <p className="eyebrow">LOCALIZATION / PROJECTS</p>
          <h2 className="page-title mt-2">Локализация ресурсов</h2>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-muted">Импортируйте локальный файл, подготовьте и проверьте перевод. Все действия выполняются Go Core напрямую через Wails.</p>
        </div>
        <Badge variant="info"><Languages size={12} /> LOCAL WORKSPACE</Badge>
      </div>

      <Card>
        <CardHeader>
          <div><CardTitle>Проект</CardTitle><p className="mt-1 text-[10px] text-muted">Исходный файл хранится как неизменяемый снимок.</p></div>
          {project && <Badge variant="success">{project.locale}</Badge>}
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 md:grid-cols-[minmax(190px,1fr)_minmax(160px,1fr)_150px_minmax(200px,1.2fr)]">
            <label className="space-y-1.5 text-[10px] font-semibold text-muted">Открытый проект
              <select aria-label="Открытый проект" value={project?.id ?? ""} onChange={(event) => void openProject(event.target.value)} className="h-10 w-full rounded-xl border border-line bg-canvas/80 px-3 text-sm font-normal text-copy outline-none focus:border-accent/50">
                <option value="">Новый импорт…</option>
                {projects.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.locale}</option>)}
              </select>
            </label>
            <label className="space-y-1.5 text-[10px] font-semibold text-muted">Название проекта
              <Input value={name} onChange={(event) => setName(event.target.value)} disabled={Boolean(project)} />
            </label>
            <label className="space-y-1.5 text-[10px] font-semibold text-muted">Язык перевода
              <Input value={locale} onChange={(event) => setLocale(event.target.value)} disabled={Boolean(project)} />
            </label>
            <div className="space-y-1.5 text-[10px] font-semibold text-muted">Файл ресурсов
              <div className="flex h-10 items-center gap-2 rounded-xl border border-line bg-canvas/80 px-3 text-xs font-normal text-muted"><FileCode2 size={14} className="shrink-0 text-teal" /><span className="truncate">{project?.filename ?? filename}</span>
                {!project && <button type="button" className="ml-auto text-accent hover:text-copy" onClick={() => sourceFile.current?.click()}>Выбрать</button>}
              </div>
            </div>
          </div>
          <input ref={sourceFile} type="file" accept=".xml,.json,text/xml,application/json" className="sr-only" onChange={(event) => void chooseSource(event)} />
          <input ref={csvFile} type="file" accept=".csv,text/csv" className="sr-only" onChange={(event) => void chooseCSV(event)} />
          {!project && (
            <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line pt-4">
              <div className="flex flex-wrap gap-2">
                <Button variant="secondary" size="sm" onClick={() => { setContent(sampleXML); setFilename("synthetic-strings.xml"); setName("Synthetic media UI"); setFormat(LocalizationFormat.LocalizationFormatAndroidXML); setError(null); setNotice("Загружен синтетический пример."); }}><FileCode2 size={13} /> Синтетический пример</Button>
                <Button variant="outline" size="sm" onClick={() => { setFilename("resources.json"); setFormat(LocalizationFormat.LocalizationFormatJSON); }}><FileJson2 size={13} /> JSON schema</Button>
              </div>
              <Button size="sm" onClick={() => void importProject()} disabled={pending || !content.trim() || !filename.trim()}><Plus size={14} /> {pending ? "Импортируем…" : "Импортировать ресурсы"}</Button>
            </div>
          )}
          {project && <div className="flex flex-wrap items-center gap-2 border-t border-line pt-4">
            <Button size="sm" onClick={() => void validate()} disabled={pending || hasUnsavedChanges}><ShieldCheck size={14} /> Проверить перевод</Button>
            <Button variant="secondary" size="sm" onClick={() => void exportCSV()} disabled={pending || hasUnsavedChanges}><Download size={13} /> CSV</Button>
            <Button variant="secondary" size="sm" onClick={() => void exportResources()} disabled={pending || hasUnsavedChanges}><Download size={13} /> Ресурсы</Button>
            <Button variant="outline" size="sm" onClick={() => csvFile.current?.click()} disabled={pending}><FileUp size={13} /> Импорт CSV</Button>
            <Button variant="outline" size="sm" onClick={() => { replaceSourceMode.current = true; sourceFile.current?.click(); }} disabled={pending}><FileCode2 size={13} /> Обновить источник</Button>
            <Button variant="secondary" size="sm" className="ml-auto" onClick={() => void buildPackage()} disabled={pending || hasUnsavedChanges}><Package size={14} /> Собрать языковой пакет</Button>
          </div>}
          {project && hasUnsavedChanges && <p className="text-[10px] text-warning">Сохраните изменения строк, прежде чем запускать проверку, экспорт или сборку пакета.</p>}
          {error && <div role="alert" className="flex items-start gap-2 rounded-xl border border-danger/20 bg-danger/[0.04] p-3 text-xs leading-5 text-danger"><AlertCircle size={14} className="mt-0.5 shrink-0" />{error}</div>}
          {notice && !error && <p role="status" className="text-xs text-teal">{notice}</p>}
        </CardContent>
      </Card>

      {report && <Card className={report.valid ? "border-accent/25 bg-accent/[0.025]" : "border-warning/25 bg-warning/[0.025]"}>
        <CardHeader><div className="flex items-center gap-2">{report.valid ? <CheckCircle2 size={16} className="text-accent" /> : <AlertCircle size={16} className="text-warning" />}<CardTitle>{report.valid ? "Проверка пройдена" : "Найдены проблемы"}</CardTitle></div><Badge variant={report.valid ? "success" : "warning"}>{report.issues?.length ?? 0} замечаний</Badge></CardHeader>
        <CardContent>{(report.issues ?? []).length ? <ul className="space-y-2">{(report.issues ?? []).map((issue, index) => <li key={`${issue.code}-${issue.resourceKey ?? ""}-${index}`} className="flex gap-2 text-xs leading-5 text-muted"><span className="font-mono text-warning">{issue.code}</span><span>{issue.resourceKey && <b className="mr-1 text-teal">{issue.resourceKey}</b>}{issue.message}</span></li>)}</ul> : <p className="text-xs text-muted">Ошибок в переводе нет. Для сборки пакета все строки должны быть утверждены.</p>}</CardContent>
      </Card>}

      {project && <Card>
        <CardHeader>
          <div><CardTitle>Строки проекта</CardTitle><p className="mt-1 text-[10px] text-muted">{project.resources?.length ?? 0} ресурсов · оригинал и перевод {project.locale}</p></div>
          <label className="relative w-full max-w-xs"><Search size={14} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-faint" /><Input aria-label="Поиск ресурсов" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Ключ, оригинал, перевод…" className="h-9 pl-9 text-xs" /></label>
        </CardHeader>
        <CardContent className="pt-4">
          <div className="mb-2 grid grid-cols-[minmax(110px,.75fr)_minmax(180px,1fr)_minmax(220px,1.2fr)_130px] gap-3 px-3 text-[9px] font-semibold uppercase tracking-[.16em] text-faint"><span>Ключ / тип</span><span>Оригинал</span><span>Перевод · {project.locale}</span><span>Статус</span></div>
          <div className="space-y-2">
            {visibleResources.map((resource) => {
              const translation = translations.get(resource.key);
              const value = drafts[resource.key] ?? translation?.targetText ?? "";
              const isDirty = value !== (translation?.targetText ?? "");
              const disabled = !resource.translatable || pending;
              return <div key={resource.key} className="grid grid-cols-[minmax(110px,.75fr)_minmax(180px,1fr)_minmax(220px,1.2fr)_130px] items-start gap-3 rounded-xl border border-line bg-canvas/35 p-3">
                <div className="min-w-0"><p className="truncate font-mono text-[10px] text-teal" title={resource.key}>{resource.key}</p><p className="mt-1 text-[9px] text-faint">{resource.kind}{!resource.translatable ? " · locked" : ""}</p></div>
                <p className="break-words pt-1 text-xs leading-5 text-copy">{resource.sourceText}</p>
                <div className="space-y-2"><Textarea aria-label={`Перевод ${resource.key}`} value={value} onChange={(event) => setDrafts((current) => ({ ...current, [resource.key]: event.target.value }))} disabled={disabled} className="min-h-16 resize-y font-sans text-xs leading-5" placeholder={resource.translatable ? "Введите перевод…" : "Не переводится"} />
                  {resource.translatable && <div className="flex gap-1.5"><Button variant="secondary" size="sm" className="h-7 rounded-lg px-2 text-[10px]" onClick={() => void saveTranslation(resource.key)} disabled={pending || !isDirty}><Check size={12} /> Сохранить</Button>{translation?.status !== "approved" && <Button size="sm" className="h-7 rounded-lg px-2 text-[10px]" onClick={() => void approveTranslation(resource.key)} disabled={pending || !value.trim() || isDirty}><CheckCircle2 size={12} /> Утвердить</Button>}</div>}
                </div>
                <div className="flex flex-col items-start gap-2">{isDirty ? <Badge variant="warning">Не сохранено</Badge> : translation ? <Badge variant={translation.status === "approved" ? "success" : translation.status === "needs_review" ? "warning" : "neutral"}>{statusLabel(translation.status)}</Badge> : <Badge variant="danger">Нет перевода</Badge>}{translation?.status === "approved" && !isDirty && <span className="text-[9px] text-faint">SHA {translation.approvedSourceSha256?.slice(0, 8)}</span>}</div>
              </div>;
            })}
            {!visibleResources.length && <div className="rounded-xl border border-dashed border-line px-4 py-10 text-center text-xs text-muted">{search ? "Совпадений нет." : "Ресурсы не найдены."}</div>}
          </div>
          {pending && <p className="mt-3 flex items-center gap-2 text-[10px] text-muted"><LoaderCircle size={12} className="animate-spin" />Сохраняем локальный проект…</p>}
        </CardContent>
      </Card>}

      {!project && !loadingProjects && <div className="flex min-h-32 flex-col items-center justify-center rounded-2xl border border-dashed border-line text-center"><div className="grid size-10 place-items-center rounded-xl border border-line bg-panel-raised text-faint"><FileUp size={17} /></div><p className="mt-3 text-xs font-semibold text-copy">Выберите файл ресурсов</p><p className="mt-1 text-[10px] text-muted">Поддерживаются Android XML и JSON schema carvion.localization/v1.</p></div>}
    </div>
  );
}

function updateTranslation(project: LocalizationProject, translation: LocalizationTranslation): LocalizationProject {
  const current = project.translations ?? [];
  const index = current.findIndex((item) => item.resourceKey === translation.resourceKey && item.locale === translation.locale);
  const translations = [...current];
  if (index < 0) translations.push(translation);
  else translations[index] = translation;
  return { ...project, translations };
}

function statusLabel(status: LocalizationTranslation["status"]): string {
  if (status === "approved") return "Утверждён";
  if (status === "needs_review") return "Проверить";
  return "Черновик";
}

async function readUTF8(file: File): Promise<string> {
  return new TextDecoder("utf-8", { fatal: true }).decode(await file.arrayBuffer());
}
