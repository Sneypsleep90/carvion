import * as ApplicationService from "../../bindings/github.com/Sneypsleep90/carvion/desktop/applicationservice";
import type { ApplicationState } from "../../bindings/github.com/Sneypsleep90/carvion/desktop/models";
import { AdapterStatus } from "../../bindings/github.com/Sneypsleep90/carvion/internal/core/domain/models";
import type {
  AdapterInfo,
  CompatibilityRequest,
  CompatibilityResult,
  DetectionResult,
  GeelyDiagnosticReport,
  GeelyDiagnosticReportRequest,
  GeelyLocalizationImportRequest,
  GeelyLocalizationImportResult,
  GeelyResearchAnalysis,
  GeelyResearchEvidence,
  GeelyResourceRequest,
  InfotainmentPlatform,
  InspectResourcesRequest,
	LocalizationArtifact,
	LocalizationFormat,
	LocalizationPackage,
	LocalizationProject,
	LocalizationProjectSummary,
	LocalizationTranslation,
	LocalizationValidationReport,
  SimulatorProfile,
  SimulatorRunResult,
  Manufacturer,
  ResourceInspection,
  VehicleIdentity,
  VehicleModel,
} from "../../bindings/github.com/Sneypsleep90/carvion/internal/core/domain/models";

export type DesktopBackend = Readonly<{
  getApplicationState: () => Promise<ApplicationState>;
  listAdapters: () => Promise<ReadonlyArray<AdapterInfo>>;
  listManufacturers: () => Promise<ReadonlyArray<Manufacturer>>;
  listVehicleModels: (manufacturerID: string) => Promise<ReadonlyArray<VehicleModel>>;
  listInfotainmentPlatforms: (manufacturerID: string) => Promise<ReadonlyArray<InfotainmentPlatform>>;
  checkCompatibility: (adapterID: string, request: CompatibilityRequest) => Promise<CompatibilityResult>;
  detectVehicle: (adapterID: string, identity: VehicleIdentity) => Promise<DetectionResult>;
  inspectResources: (adapterID: string, request: InspectResourcesRequest) => Promise<ResourceInspection>;
  listLocalizationProjects: () => Promise<ReadonlyArray<LocalizationProjectSummary>>;
  getLocalizationProject: (projectID: string) => Promise<LocalizationProject>;
  replaceLocalizationSource: (projectID: string, filename: string, content: string) => Promise<LocalizationProject>;
  importLocalizationProject: (name: string, locale: string, format: LocalizationFormat, filename: string, content: string) => Promise<LocalizationProject>;
  updateLocalizationTranslation: (projectID: string, resourceKey: string, targetText: string) => Promise<LocalizationTranslation>;
  approveLocalizationTranslation: (projectID: string, resourceKey: string) => Promise<LocalizationTranslation>;
  importLocalizationCSV: (projectID: string, content: string) => Promise<LocalizationProject>;
  exportLocalizationCSV: (projectID: string) => Promise<LocalizationArtifact>;
  exportLocalizationResources: (projectID: string) => Promise<LocalizationArtifact>;
  validateLocalizationProject: (projectID: string) => Promise<LocalizationValidationReport>;
  buildLocalizationPackage: (projectID: string) => Promise<LocalizationPackage>;
  listSimulatorProfiles: () => Promise<ReadonlyArray<SimulatorProfile>>;
  runSimulatorScenario: (profileID: string, requestID: string) => Promise<SimulatorRunResult>;
  cancelSimulatorScenario: (requestID: string) => Promise<boolean>;
  listGeelyResearchEvidence: () => Promise<ReadonlyArray<GeelyResearchEvidence>>;
  analyzeGeelyProfile: (profileJSON: string) => Promise<GeelyResearchAnalysis>;
  inspectGeelyResources: (request: GeelyResourceRequest) => Promise<ResourceInspection>;
  importGeelyLocalizationProject: (request: GeelyLocalizationImportRequest) => Promise<GeelyLocalizationImportResult>;
  buildGeelyDiagnosticReport: (request: GeelyDiagnosticReportRequest) => Promise<GeelyDiagnosticReport>;
}>;

export const desktopBackend: DesktopBackend = {
  getApplicationState: () => ApplicationService.GetApplicationState(),
  listAdapters: async () => (await ApplicationService.ListAdapters()) ?? [],
  listManufacturers: async () => (await ApplicationService.ListManufacturers()) ?? [],
  listVehicleModels: async (manufacturerID) => (await ApplicationService.ListVehicleModels(manufacturerID)) ?? [],
  listInfotainmentPlatforms: async (manufacturerID) =>
    (await ApplicationService.ListInfotainmentPlatforms(manufacturerID)) ?? [],
  checkCompatibility: (adapterID, request) => ApplicationService.CheckCompatibility(adapterID, request),
  detectVehicle: (adapterID, identity) => ApplicationService.DetectVehicle(adapterID, identity),
  inspectResources: (adapterID, request) => ApplicationService.InspectResources(adapterID, request),
  listLocalizationProjects: async () => (await ApplicationService.ListLocalizationProjects()) ?? [],
  getLocalizationProject: (projectID) => ApplicationService.GetLocalizationProject(projectID),
  replaceLocalizationSource: (projectID, filename, content) => ApplicationService.ReplaceLocalizationSource(projectID, filename, content),
  importLocalizationProject: (name, locale, format, filename, content) =>
    ApplicationService.ImportLocalizationProject(name, locale, format, filename, content),
  updateLocalizationTranslation: (projectID, resourceKey, targetText) =>
    ApplicationService.UpdateLocalizationTranslation(projectID, resourceKey, targetText),
  approveLocalizationTranslation: (projectID, resourceKey) =>
    ApplicationService.ApproveLocalizationTranslation(projectID, resourceKey),
  importLocalizationCSV: (projectID, content) => ApplicationService.ImportLocalizationCSV(projectID, content),
  exportLocalizationCSV: (projectID) => ApplicationService.ExportLocalizationCSV(projectID),
  exportLocalizationResources: (projectID) => ApplicationService.ExportLocalizationResources(projectID),
  validateLocalizationProject: (projectID) => ApplicationService.ValidateLocalizationProject(projectID),
  buildLocalizationPackage: (projectID) => ApplicationService.BuildLocalizationPackage(projectID),
  listSimulatorProfiles: async () => (await ApplicationService.ListSimulatorProfiles()) ?? [],
  runSimulatorScenario: (profileID, requestID) => ApplicationService.RunSimulatorScenario(profileID, requestID),
  cancelSimulatorScenario: (requestID) => ApplicationService.CancelSimulatorScenario(requestID),
  listGeelyResearchEvidence: async () => (await ApplicationService.ListGeelyResearchEvidence()) ?? [],
  analyzeGeelyProfile: (profileJSON) => ApplicationService.AnalyzeGeelyProfile(profileJSON),
  inspectGeelyResources: (request) => ApplicationService.InspectGeelyResources(request),
  importGeelyLocalizationProject: (request) => ApplicationService.ImportGeelyLocalizationProject(request),
  buildGeelyDiagnosticReport: (request) => ApplicationService.BuildGeelyDiagnosticReport(request),
};

export type {
  AdapterInfo,
  ApplicationState,
  CompatibilityRequest,
  CompatibilityResult,
  DetectionResult,
  GeelyDiagnosticReport,
  GeelyDiagnosticReportRequest,
  GeelyLocalizationImportRequest,
  GeelyLocalizationImportResult,
  GeelyResearchAnalysis,
  GeelyResearchEvidence,
  GeelyResourceRequest,
  InfotainmentPlatform,
  InspectResourcesRequest,
  LocalizationArtifact,
  LocalizationFormat,
  LocalizationPackage,
  LocalizationProject,
  LocalizationProjectSummary,
  LocalizationTranslation,
  LocalizationValidationReport,
  Manufacturer,
  ResourceInspection,
  SimulatorProfile,
  SimulatorRunResult,
  VehicleIdentity,
  VehicleModel,
};

export { AdapterStatus };
