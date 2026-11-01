import React from "react";
import ReactDOM from "react-dom/client";
import App from "./src/App";
import type { DesktopBackend } from "./src/lib/backend";
import {
  AdapterStatus,
  CompatibilityStatus,
  DeploymentCheckStatus,
  DeploymentMode,
  DeploymentState,
  LocalizationFormat,
  LocalizationResourceKind,
  SimulatorRunStatus,
  TranslationStatus,
  type DeploymentEvent,
  type DeploymentJob,
  type DeploymentRequest,
  type LocalizationProject,
  type LocalizationProjectSummary,
  type SimulatorProfile,
  type SimulatorRunResult,
} from "./bindings/github.com/Sneypsleep90/carvion/internal/core/domain/models";
import "./src/index.css";

let localizationProject: LocalizationProject | null = null;
const cancelledSimulatorRequests = new Set<string>();
const deploymentJobs = new Map<string, DeploymentJob>();
const deploymentListeners = new Set<(event: DeploymentEvent) => void>();
const cancelledDeploymentJobs = new Set<string>();
let nextDeploymentJobID = 1;
const deploymentProjects: ReadonlyArray<LocalizationProjectSummary> = [
  { id: "e2e-approved-project", name: "E2E · Synthetic approved localization", locale: "ru-RU", format: LocalizationFormat.LocalizationFormatAndroidXML, filename: "e2e-approved-strings.xml", sourceSha256: "e2e-source-sha256" },
  { id: "e2e-invalid-package", name: "E2E · Package integrity failure fixture", locale: "ru-RU", format: LocalizationFormat.LocalizationFormatAndroidXML, filename: "e2e-invalid-package.xml", sourceSha256: "e2e-invalid-source-sha256" },
  { id: "e2e-provider-failure", name: "E2E · Synthetic provider failure fixture", locale: "ru-RU", format: LocalizationFormat.LocalizationFormatAndroidXML, filename: "e2e-provider-failure.xml", sourceSha256: "e2e-provider-failure-source-sha256" },
];

const geelyProfileFields = (profileJSON: string) => {
  const profile = JSON.parse(profileJSON) as {
    schema: string;
    profile_id: string;
    vehicle?: { manufacturer?: string; model?: string; model_year?: number; market?: string; trim?: string };
    infotainment?: { platform?: string; os_name?: string; os_version?: string; hardware_identifier?: string; hardware_revision?: string; firmware_version?: string; firmware_build_id?: string };
  };
  return {
    schema: profile.schema,
    profileId: profile.profile_id,
    manufacturer: profile.vehicle?.manufacturer ?? "",
    model: profile.vehicle?.model ?? "",
    modelYear: profile.vehicle?.model_year ?? 0,
    market: profile.vehicle?.market ?? "",
    trim: profile.vehicle?.trim ?? "",
    platform: profile.infotainment?.platform ?? "",
    osName: profile.infotainment?.os_name ?? "",
    osVersion: profile.infotainment?.os_version ?? "",
    hardwareIdentifier: profile.infotainment?.hardware_identifier ?? "",
    hardwareRevision: profile.infotainment?.hardware_revision ?? "",
    firmwareVersion: profile.infotainment?.firmware_version ?? "",
    firmwareBuildId: profile.infotainment?.firmware_build_id ?? "",
    origin: "imported-file-unverified",
  };
};

const geelyAnalysis = (profileJSON: string) => {
  const profile = geelyProfileFields(profileJSON);
  return {
    profile,
    detection: {
      Status: CompatibilityStatus.StatusUnknown,
      Identity: { Manufacturer: profile.manufacturer, Model: profile.model, PlatformID: profile.platform },
      Reason: "Imported file values only; no vehicle connection was made.",
    },
    compatibility: {
      Status: CompatibilityStatus.StatusUnknown,
      Reason: "E2E fixture contains no exact reviewed compatibility record.",
    },
  };
};

const geelyEvidence = [{
  id: "test-evidence-only", title: "Synthetic Geely research source fixture", publisher: "CARVION E2E",
  url: "https://example.invalid/geely-research-fixture", publishedAt: "", retrievedAt: "2026-11-01",
  scope: "Только фикстура Playwright; не является техническим утверждением о мультимедиа Geely.", confidence: "test-fixture",
}];

const simulatorProfiles: ReadonlyArray<SimulatorProfile> = [
  {
    id: "demo-android-platform-a", name: "Demo Android Platform A", description: "Synthetic Android test device, profile A.", synthetic: true,
    identity: { Manufacturer: "CARVION TEST LAB", Model: "Android Demo Unit", PlatformID: "android-demo-platform" },
    hardware: { PlatformID: "android-demo-platform", Revision: "rev-a" },
    firmware: { BuildID: "demo-build-a-001", Version: "1.0.0-demo" }, expectedCompatibility: CompatibilityStatus.StatusSupported,
  },
  {
    id: "demo-android-platform-b", name: "Demo Android Platform B", description: "Synthetic Android test device, profile B.", synthetic: true,
    identity: { Manufacturer: "CARVION TEST LAB", Model: "Android Demo Unit B", PlatformID: "android-demo-platform-b" },
    hardware: { PlatformID: "android-demo-platform-b", Revision: "rev-b" },
    firmware: { BuildID: "demo-build-b-001", Version: "2.0.0-demo" }, expectedCompatibility: CompatibilityStatus.StatusSupported,
  },
  {
    id: "unknown-firmware", name: "Unknown Firmware", description: "Synthetic device with an unlisted firmware build.", synthetic: true,
    identity: { Manufacturer: "CARVION TEST LAB", Model: "Android Demo Unit", PlatformID: "android-demo-platform" },
    hardware: { PlatformID: "android-demo-platform", Revision: "rev-a" },
    firmware: { BuildID: "synthetic-unlisted-build", Version: "9.9.9-demo" }, expectedCompatibility: CompatibilityStatus.StatusUnknown,
  },
  {
    id: "incompatible-hardware", name: "Incompatible Hardware", description: "Synthetic device with an unsupported hardware revision.", synthetic: true,
    identity: { Manufacturer: "CARVION TEST LAB", Model: "Android Demo Unit", PlatformID: "android-demo-platform" },
    hardware: { PlatformID: "android-demo-platform", Revision: "rev-incompatible" },
    firmware: { BuildID: "demo-build-a-001", Version: "1.0.0-demo" }, expectedCompatibility: CompatibilityStatus.StatusUnsupported,
  },
  {
    id: "simulated-communication-failure", name: "Simulated Communication Failure", description: "Synthetic scenario that returns a controlled RPC error.", synthetic: true,
    identity: { Manufacturer: "CARVION TEST LAB", Model: "Android Demo Unit", PlatformID: "android-demo-platform" },
    hardware: { PlatformID: "android-demo-platform", Revision: "rev-a" },
    firmware: { BuildID: "synthetic-rpc-error", Version: "1.0.0-demo-error" }, expectedCompatibility: CompatibilityStatus.StatusUnknown,
  },
];

function publishDeploymentEvent(
  job: DeploymentJob,
  state: DeploymentState,
  stage: string,
  message: string,
  patch: Partial<DeploymentJob> = {},
): DeploymentJob {
  const event: DeploymentEvent = {
    jobId: job.id,
    sequence: (job.events?.length ?? 0) + 1,
    at: new Date().toISOString(),
    state,
    stage,
    message,
  };
  const updated: DeploymentJob = {
    ...job,
    ...patch,
    state,
    events: [...(job.events ?? []), event],
    updatedAt: event.at,
  };
  deploymentJobs.set(job.id, updated);
  for (const listener of deploymentListeners) listener(event);
  return updated;
}

async function runDeploymentFixture(jobID: string) {
  let job = deploymentJobs.get(jobID);
  if (!job) return;
  const pause = () => new Promise<void>((resolve) => setTimeout(resolve, 180));
  job = publishDeploymentEvent(job, DeploymentState.DeploymentStatePreflighting, "preflight.started", "Сохраненная задача перешла к предварительным проверкам.");
  await pause();
  if (cancelledDeploymentJobs.has(jobID)) return;

  if (job.mode === DeploymentMode.DeploymentModeProduction) {
    const checks = [
      { id: "package-integrity", status: DeploymentCheckStatus.DeploymentCheckPassed, required: true, message: "Структура и SHA-256 пакета проверены." },
      { id: "target-profile", status: DeploymentCheckStatus.DeploymentCheckBlocked, required: true, message: "Подтверждённая целевая платформа отсутствует." },
      { id: "compatibility", status: DeploymentCheckStatus.DeploymentCheckUnknown, required: true, message: "Точной проверенной записи совместимости нет." },
      { id: "installation-capability", status: DeploymentCheckStatus.DeploymentCheckBlocked, required: true, message: "Подтверждённый способ установки не описан." },
      { id: "result-verification", status: DeploymentCheckStatus.DeploymentCheckBlocked, required: true, message: "Критерии проверки результата на автомобиле отсутствуют." },
      { id: "recovery-capability", status: DeploymentCheckStatus.DeploymentCheckBlocked, required: true, message: "Подтверждённая процедура восстановления отсутствует." },
      { id: "bench-evidence", status: DeploymentCheckStatus.DeploymentCheckBlocked, required: true, message: "Результаты стендовых испытаний отсутствуют." },
      { id: "production-policy", status: DeploymentCheckStatus.DeploymentCheckBlocked, required: true, message: "Production Mode запрещён политикой Go Core." },
    ];
    publishDeploymentEvent(job, DeploymentState.DeploymentStateBlocked, "preflight.blocked", "Production Mode заблокирован политикой Go Core.", { preflight: { ready: false, checks } });
    return;
  }

  if (job.projectId === "e2e-invalid-package") {
    const checks = [{ id: "package-integrity", status: DeploymentCheckStatus.DeploymentCheckFailed, required: true, message: "Синтетическая E2E фикстура содержит несовпадающий SHA-256." }];
    publishDeploymentEvent(job, DeploymentState.DeploymentStateFailed, "package-integrity.failed", "Проверка целостности пакета завершилась ошибкой.", {
      preflight: { ready: false, checks }, errorMessage: "Пакет не прошёл проверку; симуляция не запускалась.",
    });
    return;
  }

  const passedChecks = [
    { id: "package-integrity", status: DeploymentCheckStatus.DeploymentCheckPassed, required: true, message: "Тестовый пакет прошёл structural и SHA-256 проверки." },
    { id: "target-profile", status: DeploymentCheckStatus.DeploymentCheckPassed, required: true, message: "Профиль выбран из синтетического каталога." },
    { id: "compatibility", status: DeploymentCheckStatus.DeploymentCheckPassed, required: true, message: "Точная совместимость подтверждена только для синтетической фикстуры." },
    { id: "provider-capability", status: DeploymentCheckStatus.DeploymentCheckPassed, required: true, message: "Provider поддерживает только симуляцию и не умеет устанавливать пакет." },
    { id: "result-verification", status: DeploymentCheckStatus.DeploymentCheckPassed, required: true, message: "Синтетическая проверка результата доступна." },
    { id: "recovery-capability", status: DeploymentCheckStatus.DeploymentCheckPassed, required: true, message: "Синтетическая проверка восстановления доступна." },
  ];
  job = publishDeploymentEvent(job, DeploymentState.DeploymentStatePreflightPassed, "preflight.passed", "Предварительные проверки синтетического сценария пройдены.", { preflight: { ready: true, checks: passedChecks } });
  await pause();
  if (cancelledDeploymentJobs.has(jobID)) return;
  job = publishDeploymentEvent(job, DeploymentState.DeploymentStateSimulating, "simulation.started", "Запущена локальная симуляция; оборудование не используется.");
  await pause();
  if (cancelledDeploymentJobs.has(jobID)) return;

  if (job.projectId === "e2e-provider-failure") {
    publishDeploymentEvent(job, DeploymentState.DeploymentStateFailed, "provider.failed", "Синтетический provider вернул контролируемую ошибку RPC.", {
      errorMessage: "Контролируемая ошибка RPC синтетического provider.",
      recovery: { available: true, simulated: true, message: "Проверка восстановления выполнена только в синтетическом сценарии." },
    });
    return;
  }

  job = publishDeploymentEvent(job, DeploymentState.DeploymentStateVerifying, "result-verification.started", "Проверяется результат симуляции.", {
    recovery: { available: true, simulated: true, message: "Синтетическая проверка восстановления; откат устройства не выполнялся." },
  });
  await pause();
  if (cancelledDeploymentJobs.has(jobID)) return;
  publishDeploymentEvent(job, DeploymentState.DeploymentStateSimulated, "simulation.completed", "Симуляция завершена; автомобиль не изменён.", {
    resultVerification: { verified: true, simulated: true, message: "Синтетический результат проверен; автомобиль не изменён." },
  });
}

const backend: DesktopBackend = {
  getApplicationState: async () => ({
    version: "0.1.0",
    coreStatus: "ready",
    databaseStatus: "connected",
    adapterStatus: "available",
    adapterCount: 2,
    message: "",
  }),
  listAdapters: async () => [
    {
      ID: "adapter-android-demo",
      Name: "CARVION Synthetic Android Demo",
      Version: "0.1.0",
      APIVersion: "v1",
      SDKVersion: "1.0.0",
      Capabilities: { Detect: true, CheckCompatibility: true, InspectResources: true },
      InputTypes: ["vehicle-observation-json", "compatibility-profile", "resource-document"],
      ResourceFormats: ["android-strings-xml"],
      Status: AdapterStatus.AdapterStatusAvailable,
    },
    {
      ID: "adapter-android-demo-alt",
      Name: "CARVION Alternate Synthetic Adapter",
      Version: "0.2.0",
      APIVersion: "v1",
      SDKVersion: "1.0.0",
      Capabilities: { Detect: false, CheckCompatibility: true, InspectResources: false },
      InputTypes: ["compatibility-profile"],
      ResourceFormats: [],
      Status: AdapterStatus.AdapterStatusUnavailable,
    },
  ],
  listManufacturers: async () => [],
  listVehicleModels: async () => [],
  listInfotainmentPlatforms: async () => [],
  checkCompatibility: async (adapterID, request) => {
    const knownDemoProfile = request.Identity.Manufacturer === "CARVION TEST LAB";
    return {
      Status: knownDemoProfile ? CompatibilityStatus.StatusSupported : CompatibilityStatus.StatusUnknown,
      Reason: `${adapterID}: ${knownDemoProfile ? "exact synthetic demo profile match" : "vehicle identity is not present in the synthetic demo profiles"}`,
    };
  },
  detectVehicle: async (_adapterID, identity) => ({
    Status: CompatibilityStatus.StatusUnknown,
    Identity: identity,
    Reason: "vehicle identity is not present in the synthetic demo profiles",
  }),
  inspectResources: async (_adapterID, request) => {
    const resources = [...request.Content.matchAll(/<string name="([^"]+)"[^>]*>(.*?)<\/string>/gs)]
      .filter((match) => !/translatable="false"/.test(match[0]))
      .map((match) => ({ ID: match[1] ?? "", Text: match[2]?.trim() ?? "" }));
    return { Resources: resources };
  },
  listLocalizationProjects: async () => localizationProject ? [{
    id: localizationProject.id,
    name: localizationProject.name,
    locale: localizationProject.locale,
    format: localizationProject.format,
    filename: localizationProject.filename,
    sourceSha256: localizationProject.sourceSha256,
  }] : [],
  listDeploymentProjects: async () => deploymentProjects,
  getLocalizationProject: async (projectID) => {
    if (!localizationProject || localizationProject.id !== projectID) throw new Error("localization project not found");
    return localizationProject;
  },
  replaceLocalizationSource: async (projectID, filename, content) => {
    if (!localizationProject || localizationProject.id !== projectID) throw new Error("localization project not found");
    const resources = parseSyntheticResources(content);
    const translations = (localizationProject.translations ?? []).map((translation) => {
      const next = resources.find((resource) => resource.key === translation.resourceKey);
      if (!next || next.sourceSha256 === translation.approvedSourceSha256) return translation;
      if (translation.status !== TranslationStatus.TranslationStatusApproved) return translation;
      const review = { ...translation, status: TranslationStatus.TranslationStatusNeedsReview };
      delete review.approvedSourceSha256;
      return review;
    });
    const updatedProject: LocalizationProject = { ...localizationProject, filename, sourceSha256: "synthetic-replacement-sha256", resources, translations };
    localizationProject = updatedProject;
    return updatedProject;
  },
  importLocalizationProject: async (name, locale, format, filename, content) => {
    const resources = parseSyntheticResources(content);
    localizationProject = {
      id: "localization-demo-project",
      name,
      locale,
      format,
      filename,
      sourceSha256: "synthetic-source-sha256",
      resources,
      translations: [],
    };
    return localizationProject;
  },
  updateLocalizationTranslation: async (projectID, resourceKey, targetText) => {
    if (!localizationProject || localizationProject.id !== projectID) throw new Error("localization project not found");
    const translation = { resourceKey, locale: localizationProject.locale, targetText, status: TranslationStatus.TranslationStatusDraft };
    localizationProject = { ...localizationProject, translations: replaceTranslation(localizationProject.translations ?? [], translation) };
    return translation;
  },
  approveLocalizationTranslation: async (projectID, resourceKey) => {
    if (!localizationProject || localizationProject.id !== projectID) throw new Error("localization project not found");
    const current = (localizationProject.translations ?? []).find((item) => item.resourceKey === resourceKey);
    if (!current?.targetText.trim()) throw new Error("translation is empty");
    const sourceHash = (localizationProject.resources ?? []).find((item) => item.key === resourceKey)?.sourceSha256;
    if (!sourceHash) throw new Error("resource not found");
    const translation = { ...current, status: TranslationStatus.TranslationStatusApproved, approvedSourceSha256: sourceHash };
    localizationProject = { ...localizationProject, translations: replaceTranslation(localizationProject.translations ?? [], translation) };
    return translation;
  },
  importLocalizationCSV: async (projectID, csv) => {
    if (!localizationProject || localizationProject.id !== projectID) throw new Error("localization project not found");
    const rows = csv.trim().split(/\r?\n/).slice(1).map((row) => row.split(","));
    const imported = rows.filter((row) => row[0]).map(([resourceKey, , targetText]) => ({ resourceKey: resourceKey ?? "", locale: localizationProject?.locale ?? "ru-RU", targetText: targetText ?? "", status: TranslationStatus.TranslationStatusDraft }));
    localizationProject = { ...localizationProject, translations: imported.reduce(replaceTranslation, localizationProject.translations ?? []) };
    return localizationProject;
  },
  exportLocalizationCSV: async (projectID) => {
    if (!localizationProject || localizationProject.id !== projectID) throw new Error("localization project not found");
    const rows = ["resource_key,source_text,target_text,locale", ...(localizationProject.resources ?? []).filter((item) => item.translatable).map((resource) => {
      const translation = (localizationProject?.translations ?? []).find((item) => item.resourceKey === resource.key);
      return [resource.key, resource.sourceText, translation?.targetText ?? "", localizationProject?.locale ?? "ru-RU"].join(",");
    })];
    return { filename: `${localizationProject.filename}.translations.csv`, contentBase64: encodeBase64(rows.join("\n") + "\n") };
  },
  exportLocalizationResources: async (projectID) => {
    if (!localizationProject || localizationProject.id !== projectID) throw new Error("localization project not found");
    return { filename: localizationProject.filename, contentBase64: encodeBase64("<resources />") };
  },
  validateLocalizationProject: async (projectID) => {
    if (!localizationProject || localizationProject.id !== projectID) throw new Error("localization project not found");
    const issues = (localizationProject.resources ?? []).filter((resource) => resource.translatable && !(localizationProject?.translations ?? []).some((translation) => translation.resourceKey === resource.key && translation.targetText.trim())).map((resource) => ({ code: "missing_translation", severity: "error", resourceKey: resource.key, message: "Для ресурса отсутствует перевод." }));
    return { valid: issues.length === 0, locale: localizationProject.locale, issues };
  },
  buildLocalizationPackage: async (projectID) => {
    if (!localizationProject || localizationProject.id !== projectID) throw new Error("localization project not found");
    const pending = (localizationProject.resources ?? []).some((resource) => resource.translatable && !(localizationProject?.translations ?? []).some((translation) => translation.resourceKey === resource.key && translation.status === TranslationStatus.TranslationStatusApproved));
    if (pending) throw new Error("all translations must be approved");
    return { filename: `${localizationProject.name}-${localizationProject.locale}.carvion.zip`, sha256: "synthetic-package-hash", size: 10, contentBase64: encodeBase64("demo-zip") };
  },
  listSimulatorProfiles: async () => simulatorProfiles,
  runSimulatorScenario: async (profileID, requestID) => {
    if (profileID !== "demo-android-platform-b") return createSimulatorRun(profileID);
    await new Promise((resolve) => setTimeout(resolve, 300));
    if (cancelledSimulatorRequests.has(requestID)) return createCancelledSimulatorRun(profileID);
    return createSimulatorRun(profileID);
  },
  cancelSimulatorScenario: async (requestID) => {
    cancelledSimulatorRequests.add(requestID);
    return true;
  },
  listGeelyResearchEvidence: async () => geelyEvidence,
  analyzeGeelyProfile: async (profileJSON) => geelyAnalysis(profileJSON),
  inspectGeelyResources: async (request) => ({
    Resources: parseSyntheticResources(request.content)
      .filter((resource) => resource.translatable && resource.present)
      .map((resource) => ({ ID: resource.key, Text: resource.sourceText })),
  }),
  importGeelyLocalizationProject: async (request) => {
    const resources = parseSyntheticResources(request.content);
    localizationProject = {
      id: "geely-research-e2e-project",
      name: request.name,
      locale: request.locale,
      format: request.format,
      filename: request.filename,
      sourceSha256: "synthetic-geely-source-sha256",
      resources,
      translations: [],
    };
    return {
      project: localizationProject,
      analysis: geelyAnalysis(request.profileJson),
      inspection: { Resources: resources.filter((resource) => resource.translatable).map((resource) => ({ ID: resource.key, Text: resource.sourceText })) },
    };
  },
  buildGeelyDiagnosticReport: async (request) => {
    const profile = geelyProfileFields(request.profileJson);
    const resource = request.resource ? {
      filename: request.resource.filename.split(/[\\/]/).pop() ?? request.resource.filename,
      format: request.resource.format,
      sha256: "synthetic-resource-hash",
      keys: parseSyntheticResources(request.resource.content).filter((item) => item.translatable).map((item) => item.key),
    } : undefined;
    return {
      filename: "carvion-geely-diagnostic-report.json",
      content: JSON.stringify({ schema: "test-report-only", profile, compatibility: geelyAnalysis(request.profileJson).compatibility, ...(resource ? { resource } : {}) }, null, 2),
    };
  },
  listDeploymentJobs: async (limit) => [...deploymentJobs.values()]
    .sort((left, right) => (right.createdAt ?? "").localeCompare(left.createdAt ?? ""))
    .slice(0, limit)
    .map(({ id, projectId, targetProfileId, mode, packageSha256, state, createdAt, updatedAt }) => ({
      id, projectId, ...(targetProfileId ? { targetProfileId } : {}), mode, packageSha256, state,
      createdAt: createdAt ?? "", updatedAt: updatedAt ?? "",
    })),
  startDeploymentJob: async (request: DeploymentRequest) => {
    const now = new Date().toISOString();
    const id = `e2e-deployment-${nextDeploymentJobID++}`;
    const job: DeploymentJob = {
      id,
      projectId: request.projectId,
      ...(request.targetProfileId ? { targetProfileId: request.targetProfileId } : {}),
      mode: request.mode,
      ...(request.mode === DeploymentMode.DeploymentModeTest ? { providerId: "e2e-synthetic-provider" } : {}),
      packageSha256: `e2e-package-${id}`,
      state: DeploymentState.DeploymentStateQueued,
      preflight: { ready: false, checks: [] },
      resultVerification: { verified: false, simulated: false, message: "" },
      recovery: { available: false, simulated: false, message: "" },
      events: [],
      createdAt: now,
      updatedAt: now,
    };
    deploymentJobs.set(id, job);
    void runDeploymentFixture(id);
    return job;
  },
  getDeploymentJob: async (jobID) => {
    const job = deploymentJobs.get(jobID);
    if (!job) throw new Error("deployment job not found");
    return job;
  },
  cancelDeploymentJob: async (jobID) => {
    const job = deploymentJobs.get(jobID);
    if (!job) throw new Error("deployment job not found");
    if (![DeploymentState.DeploymentStateBlocked, DeploymentState.DeploymentStateSimulated, DeploymentState.DeploymentStateFailed, DeploymentState.DeploymentStateCancelled].includes(job.state)) {
      cancelledDeploymentJobs.add(jobID);
      return publishDeploymentEvent(job, DeploymentState.DeploymentStateCancelled, "deployment.cancelled", "Тестовый прогон отменён пользователем; автомобиль не изменён.", { errorMessage: "Тестовый прогон отменён пользователем." });
    }
    return job;
  },
  onDeploymentProgress: (listener) => {
    deploymentListeners.add(listener);
    return () => deploymentListeners.delete(listener);
  },
};

function createSimulatorRun(profileID: string): SimulatorRunResult {
  const profile = simulatorProfiles.find((item) => item.id === profileID);
  if (!profile) throw new Error("unknown simulator profile");
  const event = (step: string, status: string, message: string) => ({ at: "2026-11-01T12:00:00Z", step, status, message });
  if (profileID === "simulated-communication-failure") {
    return {
      runId: "sim-run-failure", profile, adapterId: "adapter-android-demo", status: SimulatorRunStatus.SimulatorRunFailed,
      compatibility: { Status: CompatibilityStatus.StatusUnknown, Reason: "Адаптер не ответил на тестовый запрос." },
      package: { filename: "", sha256: "", size: 0, contentBase64: "" }, packageVerified: false, errorMessage: "simulated adapter communication failure",
      events: [event("scenario.started", "completed", "Запущен синтетический сценарий."), event("adapter.detect", "failed", "simulated adapter communication failure")],
    };
  }

  const compatibility = {
    Status: profile.expectedCompatibility,
    Reason: profile.expectedCompatibility === CompatibilityStatus.StatusSupported ? "Точное совпадение тестовой сборки и аппаратной ревизии."
      : profile.expectedCompatibility === CompatibilityStatus.StatusUnsupported ? "Аппаратная ревизия несовместима с этой сборкой."
        : "Сборка отсутствует в локальном тестовом каталоге.",
  };
  if (profile.expectedCompatibility !== CompatibilityStatus.StatusSupported) {
    const status = profile.expectedCompatibility === CompatibilityStatus.StatusUnsupported ? SimulatorRunStatus.SimulatorRunUnsupported : SimulatorRunStatus.SimulatorRunUnknown;
    const label = profile.expectedCompatibility === CompatibilityStatus.StatusUnsupported ? "unsupported" : "unknown";
    return {
      runId: `sim-run-${profileID}`, profile, adapterId: "adapter-android-demo", status, compatibility, packageVerified: false,
      package: { filename: "", sha256: "", size: 0, contentBase64: "" },
      events: [event("scenario.started", "completed", "Запущен синтетический сценарий."), event("device.detected", "completed", "Обнаружено виртуальное тестовое устройство."), event("adapter.selected", "completed", "Выбран локальный demo adapter."), event("compatibility.checked", label, compatibility.Reason), event("scenario.stopped", label, "Пакет локализации не создавался.")],
    };
  }

  return {
    runId: `sim-run-${profileID}`, profile, adapterId: "adapter-android-demo", status: SimulatorRunStatus.SimulatorRunCompleted,
    compatibility, localizationProjectId: "synthetic-localization-project", packageVerified: true,
    package: { filename: "CARVION-Simulator-ru-RU.carvion.zip", sha256: "synthetic-e2e-package-hash", size: 18, contentBase64: encodeBase64("synthetic package") },
    events: [
      event("scenario.started", "completed", "Запущен синтетический сценарий."),
      event("device.detected", "completed", "Обнаружено виртуальное тестовое устройство."),
      event("adapter.selected", "completed", "Выбран локальный demo adapter."),
      event("compatibility.checked", "supported", compatibility.Reason),
      event("resources.inspected", "completed", "Прочитаны тестовые строки Android XML."),
      event("localization.imported", "completed", "Импортированы тестовые китайские строки."),
      event("translations.prepared", "completed", "Подготовлены синтетические русские переводы."),
      event("translations.validated", "completed", "Переводы прошли проверку."),
      event("package.built", "completed", "Собран тестовый CARVION пакет."),
      event("package.verified", "completed", "SHA-256 и non-OEM manifest проверены."),
      event("scenario.completed", "completed", "Установка не выполнялась."),
    ],
  };
}

function createCancelledSimulatorRun(profileID: string): SimulatorRunResult {
  const profile = simulatorProfiles.find((item) => item.id === profileID);
  if (!profile) throw new Error("unknown simulator profile");
  return {
    runId: `sim-run-cancelled-${profileID}`, profile, adapterId: "adapter-android-demo", status: SimulatorRunStatus.SimulatorRunFailed,
    compatibility: { Status: CompatibilityStatus.StatusUnknown, Reason: "Сценарий отменён пользователем." },
    package: { filename: "", sha256: "", size: 0, contentBase64: "" }, packageVerified: false,
    errorMessage: "context canceled",
    events: [
      { at: "2026-11-01T12:00:00Z", step: "scenario.started", status: "completed", message: "Запущен синтетический сценарий." },
      { at: "2026-11-01T12:00:01Z", step: "scenario.cancelled", status: "failed", message: "context canceled" },
    ],
  };
}

function parseSyntheticResources(content: string): NonNullable<LocalizationProject["resources"]> {
  const resources: NonNullable<LocalizationProject["resources"]> = [];
  if (content.trimStart().startsWith("{")) {
    const document = JSON.parse(content) as { schema?: string; resources?: ReadonlyArray<{ key?: string; kind?: string; text?: string; translatable?: boolean }> };
    if (document.schema !== "carvion.localization/v1" || !Array.isArray(document.resources)) throw new Error("invalid synthetic localization JSON");
    for (const item of document.resources) {
      if (!item.key) continue;
      resources.push({ key: item.key, kind: LocalizationResourceKind.ResourceKindString, sourceText: item.text ?? "", sourceSha256: `hash-${item.key}-${item.text ?? ""}`, order: resources.length, translatable: item.translatable !== false, formatted: true, present: true });
    }
    return resources;
  }
  for (const match of content.matchAll(/<string\b([^>]*)>([\s\S]*?)<\/string>/g)) {
    const attrs = match[1] ?? "";
    const key = attrs.match(/name="([^"]+)"/)?.[1] ?? "";
    if (!key) continue;
    const sourceText = decodeEntities(match[2] ?? "");
    resources.push({ key, kind: LocalizationResourceKind.ResourceKindString, sourceText, sourceSha256: `hash-${key}-${sourceText}`, order: resources.length, translatable: !/translatable="false"/.test(attrs), formatted: true, present: true });
  }
  for (const match of content.matchAll(/<string-array\b[^>]*name="([^"]+)"[^>]*>([\s\S]*?)<\/string-array>/g)) {
    const group = match[1] ?? "";
    for (const [index, item] of [...(match[2] ?? "").matchAll(/<item\b[^>]*>([\s\S]*?)<\/item>/g)].entries()) {
      resources.push({ key: `${group}[${index}]`, groupKey: group, kind: LocalizationResourceKind.ResourceKindStringArrayItem, sourceText: decodeEntities(item[1] ?? ""), sourceSha256: `hash-${group}-${index}`, order: resources.length, itemIndex: index, translatable: true, formatted: true, present: true });
    }
  }
  for (const match of content.matchAll(/<plurals\b[^>]*name="([^"]+)"[^>]*>([\s\S]*?)<\/plurals>/g)) {
    const group = match[1] ?? "";
    for (const item of (match[2] ?? "").matchAll(/<item\b[^>]*quantity="([^"]+)"[^>]*>([\s\S]*?)<\/item>/g)) {
      const category = item[1] ?? "";
      resources.push({ key: `${group}#${category}`, groupKey: group, kind: LocalizationResourceKind.ResourceKindPluralItem, sourceText: decodeEntities(item[2] ?? ""), sourceSha256: `hash-${group}-${category}`, order: resources.length, pluralCategory: category, translatable: true, formatted: true, present: true });
    }
  }
  return resources;
}

function replaceTranslation(translations: NonNullable<LocalizationProject["translations"]>, next: NonNullable<LocalizationProject["translations"]>[number]) {
  const index = translations.findIndex((item) => item.resourceKey === next.resourceKey && item.locale === next.locale);
  if (index < 0) return [...translations, next];
  return translations.map((item, itemIndex) => itemIndex === index ? next : item);
}

function decodeEntities(text: string) {
  return text.replace(/&amp;/g, "&").replace(/&lt;/g, "<").replace(/&gt;/g, ">").replace(/&quot;/g, '"').replace(/&apos;/g, "'");
}

function encodeBase64(value: string) {
  return btoa(String.fromCharCode(...new TextEncoder().encode(value)));
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App backend={backend} />
  </React.StrictMode>,
);
