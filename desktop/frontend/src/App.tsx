import { useCallback, useEffect, useState } from "react";
import type { AdapterInfo, ApplicationState, DesktopBackend } from "./lib/backend";
import { errorMessage } from "./lib/utils";
import { sectionDescription, type SectionId } from "./navigation";
import { Sidebar } from "./components/sidebar";
import { Topbar } from "./components/topbar";
import { AdaptersPage } from "./features/adapters-page";
import { DashboardPage } from "./features/dashboard-page";
import { DiagnosticsPage } from "./features/diagnostics-page";
import { LocalizationPage } from "./features/localization-page";
import { LanguagePackagesPage, OperationHistoryPage, SettingsPage } from "./features/planned-pages";
import { VehiclesPage } from "./features/vehicles-page";

type AppProps = Readonly<{ backend: DesktopBackend }>;

export default function App({ backend }: AppProps) {
  const [section, setSection] = useState<SectionId>("dashboard");
  const [applicationState, setApplicationState] = useState<ApplicationState | null>(null);
  const [stateLoading, setStateLoading] = useState(true);
  const [stateError, setStateError] = useState<string | null>(null);
  const [adapters, setAdapters] = useState<ReadonlyArray<AdapterInfo> | null>(null);
  const [adaptersLoading, setAdaptersLoading] = useState(true);
  const [adaptersError, setAdaptersError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setStateLoading(true);
    setAdaptersLoading(true);
    setStateError(null);
    setAdaptersError(null);

    const [stateResult, adaptersResult] = await Promise.allSettled([
      backend.getApplicationState(),
      backend.listAdapters(),
    ]);

    if (stateResult.status === "fulfilled") {
      setApplicationState(stateResult.value);
    } else {
      setStateError(errorMessage(stateResult.reason));
    }
    if (adaptersResult.status === "fulfilled") {
      setAdapters(adaptersResult.value);
    } else {
      setAdaptersError(errorMessage(adaptersResult.reason));
    }
    setStateLoading(false);
    setAdaptersLoading(false);
  }, [backend]);

  const refreshAdapters = useCallback(async () => {
    setAdaptersLoading(true);
    setAdaptersError(null);
    try {
      setAdapters(await backend.listAdapters());
    } catch (cause: unknown) {
      setAdaptersError(errorMessage(cause));
    } finally {
      setAdaptersLoading(false);
    }
  }, [backend]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (section === "adapters") void refreshAdapters();
  }, [section, refreshAdapters]);

  function renderSection() {
    switch (section) {
      case "dashboard":
        return (
          <DashboardPage
            state={applicationState}
            stateLoading={stateLoading}
            stateError={stateError}
            adapters={adapters}
            adaptersLoading={adaptersLoading}
            adaptersError={adaptersError}
            onNavigate={setSection}
          />
        );
      case "vehicles":
        return <VehiclesPage backend={backend} />;
      case "diagnostics":
        return <DiagnosticsPage backend={backend} adapters={adapters ?? []} />;
      case "localization":
        return <LocalizationPage backend={backend} adapters={adapters ?? []} />;
      case "language-packages":
        return <LanguagePackagesPage onNavigate={setSection} />;
      case "adapters":
        return <AdaptersPage adapters={adapters} loading={adaptersLoading} error={adaptersError} state={applicationState} onRefresh={() => void refreshAdapters()} />;
      case "history":
        return <OperationHistoryPage />;
      case "settings":
        return <SettingsPage state={applicationState} />;
      default: {
        const unreachable: never = section;
        return unreachable;
      }
    }
  }

  return (
    <div className="studio-shell flex min-h-screen bg-canvas text-copy">
      <Sidebar activeSection={section} applicationState={applicationState} onNavigate={setSection} />
      <div className="studio-main flex min-w-0 flex-1 flex-col">
        <Topbar section={section} state={applicationState} loading={stateLoading || adaptersLoading} onRefresh={() => void refresh()} />
        <main className="studio-content w-full flex-1 px-5 py-6 md:px-7 md:py-7 xl:px-9">
          <div className="mx-auto max-w-[1480px]">
            <p className="sr-only" aria-live="polite">{sectionDescription[section]}</p>
            {renderSection()}
          </div>
        </main>
      </div>
    </div>
  );
}
