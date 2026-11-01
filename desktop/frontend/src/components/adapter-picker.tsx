import { AdapterStatus, type AdapterInfo } from "../lib/backend";

type AdapterPickerProps = Readonly<{
  adapters: ReadonlyArray<AdapterInfo>;
  selectedAdapterID: string;
  onChange: (adapterID: string) => void;
  capability: "CheckCompatibility" | "InspectResources";
  label: string;
  resourceFormat?: string;
}>;

export function AdapterPicker({ adapters, selectedAdapterID, onChange, capability, label, resourceFormat }: AdapterPickerProps) {
  const eligible = adapters.filter((adapter) =>
    adapter.Capabilities[capability] && (!resourceFormat || (adapter.ResourceFormats ?? []).includes(resourceFormat)),
  );

  return (
    <label className="block min-w-52 space-y-1.5">
      <span className="text-[10px] font-semibold text-muted">{label}</span>
      <select
        aria-label={label}
        value={eligible.some((adapter) => adapter.ID === selectedAdapterID) ? selectedAdapterID : ""}
        onChange={(event) => onChange(event.target.value)}
        disabled={eligible.length === 0}
        className="h-9 w-full rounded-xl border border-line bg-canvas px-3 text-xs text-copy outline-none focus:border-teal/50 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {eligible.length === 0 ? <option value="">Нет подходящего адаптера</option> : null}
        {eligible.map((adapter) => (
          <option key={adapter.ID} value={adapter.ID}>
            {adapter.Name} · {adapter.Status === AdapterStatus.AdapterStatusAvailable ? "доступен" : "ошибка последнего вызова"}
          </option>
        ))}
      </select>
    </label>
  );
}

export function adaptersWithCapability(
  adapters: ReadonlyArray<AdapterInfo>,
  capability: "CheckCompatibility" | "InspectResources",
  resourceFormat?: string,
) {
  return adapters.filter((adapter) =>
    adapter.Capabilities[capability] && (!resourceFormat || (adapter.ResourceFormats ?? []).includes(resourceFormat)),
  );
}
