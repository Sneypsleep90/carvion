import * as ApplicationService from "../../bindings/github.com/Sneypsleep90/carvion/desktop/applicationservice";
import type { ApplicationState } from "../../bindings/github.com/Sneypsleep90/carvion/desktop/models";
import { AdapterStatus } from "../../bindings/github.com/Sneypsleep90/carvion/internal/core/domain/models";
import type {
  AdapterInfo,
  CompatibilityRequest,
  CompatibilityResult,
  DetectionResult,
  InfotainmentPlatform,
  InspectResourcesRequest,
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
};

export type {
  AdapterInfo,
  ApplicationState,
  CompatibilityRequest,
  CompatibilityResult,
  DetectionResult,
  InfotainmentPlatform,
  InspectResourcesRequest,
  Manufacturer,
  ResourceInspection,
  VehicleIdentity,
  VehicleModel,
};

export { AdapterStatus };
