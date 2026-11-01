import { Bell, ChevronRight, CircleDot, RefreshCw, Search } from "lucide-react";
import type { ApplicationState } from "../lib/backend";
import type { SectionId } from "../navigation";
import { sectionTitles } from "../navigation";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";

type TopbarProps = Readonly<{
  section: SectionId;
  state: ApplicationState | null;
  loading: boolean;
  onRefresh: () => void;
}>;

export function Topbar({ section, state, loading, onRefresh }: TopbarProps) {
  const ready = state?.coreStatus === "ready";

  return (
    <header className="topbar flex h-[66px] shrink-0 items-center justify-between border-b border-line px-7">
      <div className="flex min-w-0 items-center gap-2 text-xs text-muted">
        <span className="font-medium text-faint">CARVION</span>
        <ChevronRight size={13} className="text-faint" aria-hidden="true" />
        <span className="truncate font-semibold text-copy">{sectionTitles[section]}</span>
      </div>
      <div className="flex items-center gap-3">
        <div className="hidden items-center gap-2 rounded-full border border-line bg-panel/75 px-3 py-1.5 sm:flex">
          <CircleDot size={13} className={ready ? "text-accent" : "text-warning"} aria-hidden="true" />
          <span className="text-[10px] font-semibold uppercase tracking-[0.12em] text-muted">Локальная лаборатория</span>
          <Badge variant="neutral" className="border-0 bg-transparent px-0 py-0 text-[9px]">TEST</Badge>
        </div>
        <Button variant="ghost" size="icon" className="size-8" aria-label="Поиск появится позднее" disabled>
          <Search size={16} />
        </Button>
        <Button variant="ghost" size="icon" className="relative size-8" aria-label="Уведомлений нет" disabled>
          <Bell size={16} />
        </Button>
        <span className="mx-0.5 h-6 w-px bg-line" aria-hidden="true" />
        <Button variant="secondary" size="sm" onClick={onRefresh} disabled={loading}>
          <RefreshCw size={13} className={loading ? "animate-spin" : ""} aria-hidden="true" />
          Обновить
        </Button>
      </div>
    </header>
  );
}
