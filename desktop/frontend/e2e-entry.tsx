import React from "react";
import ReactDOM from "react-dom/client";
import App from "./src/App";
import type { DesktopBackend } from "./src/lib/backend";
import { AdapterStatus, CompatibilityStatus } from "./bindings/github.com/Sneypsleep90/carvion/internal/core/domain/models";
import "./src/index.css";

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
};

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App backend={backend} />
  </React.StrictMode>,
);
