import { useState } from "react";
import { Activity, ArrowUpRight, ChevronDown, CircleHelp, Gauge, type LucideIcon } from "lucide-react";
import { navigationGroups, type SectionId } from "../navigation";
import type { ApplicationState } from "../lib/backend";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { cn } from "../lib/utils";

type SidebarProps = Readonly<{
  activeSection: SectionId;
  applicationState: ApplicationState | null;
  onNavigate: (section: SectionId) => void;
}>;

export function Sidebar({ activeSection, applicationState, onNavigate }: SidebarProps) {
  const [expanded, setExpanded] = useState(true);
  const coreReady = applicationState?.coreStatus === "ready";

  return (
    <aside className="sidebar-shell flex w-[248px] shrink-0 flex-col border-r border-line bg-sidebar px-3 pb-3 pt-4">
      <div className="flex h-12 items-center gap-3 px-2.5">
        <div className="brand-mark grid size-9 place-items-center rounded-xl bg-accent text-accent-foreground shadow-[0_4px_18px_rgba(184,239,112,0.22)]">
          <Gauge size={20} strokeWidth={2.4} aria-hidden="true" />
        </div>
        <div className="min-w-0">
          <h1 className="truncate text-[15px] font-bold tracking-[-0.03em] text-copy">CARVION Studio</h1>
          <p className="mt-0.5 text-[9px] font-semibold uppercase tracking-[0.2em] text-faint">Vehicle Systems Lab</p>
        </div>
      </div>

      <div className="mt-7 px-2.5">
        <button
          type="button"
          className="group flex w-full items-center justify-between rounded-lg py-1 text-left"
          aria-expanded={expanded}
          onClick={() => setExpanded((value) => !value)}
        >
          <span className="text-[10px] font-bold uppercase tracking-[0.18em] text-faint">Рабочее пространство</span>
          <ChevronDown size={14} className={cn("text-faint transition", expanded && "rotate-180")} aria-hidden="true" />
        </button>
      </div>

      {expanded && (
        <nav aria-label="Основная навигация" className="mt-3 flex-1 space-y-6">
          {navigationGroups.map((group) => (
            <div key={group.label}>
              <p className="px-2.5 pb-2 text-[9px] font-bold tracking-[0.18em] text-faint/80">{group.label}</p>
              <div className="space-y-1">
                {group.items.map((item) => (
                  <SidebarItem
                    key={item.id}
                    icon={item.icon}
                    label={item.label}
                    active={activeSection === item.id}
                    onClick={() => onNavigate(item.id)}
                  />
                ))}
              </div>
            </div>
          ))}
        </nav>
      )}

      <div className="mt-auto space-y-3">
        <div className="rounded-xl border border-line bg-panel/70 p-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-semibold text-copy">
              <span className={cn("size-1.5 rounded-full", coreReady ? "bg-accent shadow-[0_0_10px_rgba(184,239,112,0.6)]" : "bg-warning")} />
              Локальное ядро
            </div>
            <Badge variant={coreReady ? "success" : "warning"} className="px-2 py-1">
              {coreReady ? "активно" : "проверка"}
            </Badge>
          </div>
          <p className="mt-2 text-[10px] leading-4 text-muted">Автомобиль не подключён. Доступны локальные тестовые данные.</p>
          <div className="mt-3 flex items-center gap-1.5 border-t border-line pt-2.5 text-[10px] text-faint">
            <Activity size={12} aria-hidden="true" />
            {applicationState ? `Core ${applicationState.version}` : "Состояние загружается"}
          </div>
        </div>
        <div className="flex items-center justify-between px-2.5">
          <span className="text-[10px] text-faint">CARVION Studio · 0.1.0</span>
          <Button variant="ghost" size="icon" className="size-7" aria-label="Справка недоступна в этом этапе" disabled>
            <CircleHelp size={15} />
          </Button>
        </div>
      </div>
    </aside>
  );
}

function SidebarItem({
  icon: Icon,
  label,
  active,
  onClick,
}: Readonly<{ icon: LucideIcon; label: string; active: boolean; onClick: () => void }>) {
  return (
    <button
      type="button"
      aria-current={active ? "page" : undefined}
      onClick={onClick}
      className={cn(
        "group relative flex h-10 w-full items-center gap-3 rounded-xl px-2.5 text-left text-[12px] font-medium transition",
        active ? "bg-panel-raised text-copy shadow-[inset_0_0_0_1px_rgba(184,239,112,0.08)]" : "text-muted hover:bg-panel/80 hover:text-copy",
      )}
    >
      {active && <span className="absolute inset-y-2.5 left-0 w-0.5 rounded-r bg-accent" />}
      <Icon size={16} strokeWidth={active ? 2.2 : 1.8} className={cn("shrink-0", active ? "text-accent" : "text-faint group-hover:text-copy")} aria-hidden="true" />
      <span className="truncate">{label}</span>
      {label === "Адаптеры" && <ArrowUpRight size={12} className="ml-auto text-faint opacity-0 transition group-hover:opacity-100" aria-hidden="true" />}
    </button>
  );
}
