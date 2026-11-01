import { useEffect, useState, type ChangeEvent } from "react";
import { AlertCircle, FileCode2, FileUp, Languages, ScanSearch } from "lucide-react";
import type { AdapterInfo, DesktopBackend, ResourceInspection } from "../lib/backend";
import { errorMessage } from "../lib/utils";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Textarea } from "../components/ui/textarea";
import { AdapterPicker, adaptersWithCapability } from "../components/adapter-picker";

const syntheticXML = `<resources>\n  <string name="welcome_title">CARVION Studio</string>\n  <string name="station_subtitle">Локальная инженерная среда</string>\n  <string name="internal_route" translatable="false">platform-debug</string>\n</resources>`;

type LocalizationPageProps = Readonly<{ backend: DesktopBackend; adapters: ReadonlyArray<AdapterInfo> }>;

export function LocalizationPage({ backend, adapters }: LocalizationPageProps) {
  const [selectedAdapterID, setSelectedAdapterID] = useState("");
  const [content, setContent] = useState("");
  const [inspection, setInspection] = useState<ResourceInspection | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const eligibleAdapters = adaptersWithCapability(adapters, "InspectResources", "android-strings-xml");

  useEffect(() => {
    if (!eligibleAdapters.some((adapter) => adapter.ID === selectedAdapterID)) {
      setSelectedAdapterID(eligibleAdapters[0]?.ID ?? "");
    }
  }, [adapters, selectedAdapterID]);

  function loadSample() {
    setContent(syntheticXML);
    setInspection(null);
    setError(null);
  }

  async function inspect() {
    setPending(true);
    setError(null);
    setInspection(null);
    try {
      setInspection(await backend.inspectResources(selectedAdapterID, {
        Format: "android-strings-xml",
        Filename: "strings.xml",
        Content: content,
      }));
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }

  async function loadFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    setError(null);
    try {
      setContent(await file.text());
      setInspection(null);
    } catch (cause: unknown) {
      setError(errorMessage(cause));
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <p className="eyebrow">РЕСУРСЫ / ANDROID</p>
        <h2 className="page-title mt-2">Инспекция строк</h2>
        <p className="mt-2 max-w-2xl text-sm leading-6 text-muted">Передайте XML тестовому адаптеру через Go Core. Исходный файл остаётся локальным и не изменяется.</p>
      </div>

      <Card className="border-teal/15 bg-teal/[0.025]">
        <CardContent className="flex items-start gap-3 p-4">
          <Languages size={17} className="mt-0.5 shrink-0 text-teal" />
          <p className="text-xs leading-5 text-muted">Step 2 выполняет только чтение. Редактирование, перевод, сохранение проектов и экспорт языковых пакетов относятся к следующему этапу.</p>
          <Badge variant="info" className="ml-auto shrink-0">READ ONLY</Badge>
        </CardContent>
      </Card>

      <div className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3"><div className="grid size-9 place-items-center rounded-xl border border-line bg-panel-raised text-teal"><FileCode2 size={17} /></div><div><CardTitle>strings.xml</CardTitle><p className="mt-1 text-[10px] text-muted">Формат Android resources</p></div></div>
            <Badge variant="neutral">LOCAL FILE</Badge>
          </CardHeader>
          <CardContent>
            <label htmlFor="android-xml" className="mb-2 block text-[10px] font-semibold text-muted">XML-ресурсы Android</label>
            <div className="mb-3">
              <AdapterPicker adapters={adapters} selectedAdapterID={selectedAdapterID} onChange={setSelectedAdapterID} capability="InspectResources" resourceFormat="android-strings-xml" label="Адаптер ресурсов" />
            </div>
            <Textarea id="android-xml" value={content} onChange={(event) => { setContent(event.target.value); setInspection(null); }} placeholder="Вставьте содержимое <resources>…</resources>" className="min-h-[330px]" spellCheck={false} />
            <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
              <div className="flex gap-2">
                <Button variant="secondary" size="sm" onClick={loadSample}><FileCode2 size={13} /> Загрузить тестовый XML</Button>
                <label className="inline-flex h-8 cursor-pointer items-center gap-2 rounded-xl border border-line px-3 text-xs font-medium text-muted hover:bg-panel-raised hover:text-copy">
                  <FileUp size={13} /> Выбрать файл
                  <input type="file" accept=".xml,text/xml,application/xml" className="sr-only" onChange={loadFile} />
                </label>
              </div>
              <Button size="sm" onClick={() => void inspect()} disabled={pending || !content.trim() || !selectedAdapterID}>
                <ScanSearch size={14} /> {pending ? "Читаем…" : "Проверить строки"}
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div><CardTitle>Найденные строки</CardTitle><p className="mt-1 text-[10px] text-muted">Только ресурсы, отмеченные как переводимые.</p></div>
            <Badge variant={inspection ? "success" : "neutral"}>{inspection?.Resources?.length ?? 0} найдено</Badge>
          </CardHeader>
          <CardContent>
            {error && <div role="alert" className="mb-3 flex items-start gap-2 rounded-xl border border-danger/20 bg-danger/[0.04] p-3 text-xs leading-5 text-danger"><AlertCircle size={14} className="mt-0.5 shrink-0" />{error}</div>}
            {inspection ? (
              inspection.Resources?.length ? (
                <div className="space-y-2">
                  {inspection.Resources.map((resource) => (
                    <div key={resource.ID} className="grid grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)] gap-3 rounded-xl border border-line bg-canvas/35 p-3">
                      <span className="truncate font-mono text-[10px] text-teal">{resource.ID}</span>
                      <span className="text-xs text-copy">{resource.Text}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="rounded-xl border border-dashed border-line px-4 py-10 text-center text-xs text-muted">Переводимые строки не найдены.</div>
              )
            ) : (
              <div className="empty-state flex min-h-[330px] flex-col items-center justify-center rounded-xl border border-dashed border-line px-5 text-center">
                <div className="grid size-11 place-items-center rounded-2xl border border-line bg-panel-raised text-faint"><ScanSearch size={19} /></div>
                <p className="mt-3 text-xs font-semibold text-copy">Результаты появятся здесь</p>
                <p className="mt-1 max-w-xs text-[10px] leading-5 text-muted">Загрузите синтетический пример или локальный XML, затем запустите чтение.</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
