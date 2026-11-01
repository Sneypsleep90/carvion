import {
  CarFront,
  ChartNoAxesCombined,
  Clock3,
  Languages,
  PackageOpen,
  PlugZap,
  ScanLine,
  Waypoints,
  Settings2,
  SearchCode,
  type LucideIcon,
} from "lucide-react";

export type SectionId =
  | "dashboard"
  | "vehicles"
  | "diagnostics"
  | "localization"
  | "language-packages"
  | "adapters"
  | "geely-research"
  | "simulator"
  | "history"
  | "settings";

export type NavigationItem = Readonly<{ id: SectionId; label: string; icon: LucideIcon }>;

export const navigationGroups: ReadonlyArray<Readonly<{ label: string; items: ReadonlyArray<NavigationItem> }>> = [
  {
    label: "РАБОЧЕЕ МЕСТО",
    items: [
      { id: "dashboard", label: "Обзор", icon: ChartNoAxesCombined },
      { id: "vehicles", label: "Автомобили", icon: CarFront },
      { id: "diagnostics", label: "Диагностика", icon: ScanLine },
      { id: "localization", label: "Локализация", icon: Languages },
      { id: "language-packages", label: "Языковые пакеты", icon: PackageOpen },
      { id: "simulator", label: "Симулятор", icon: Waypoints },
    ],
  },
  {
    label: "СИСТЕМА",
    items: [
      { id: "adapters", label: "Адаптеры", icon: PlugZap },
      { id: "geely-research", label: "Исследование Geely", icon: SearchCode },
      { id: "history", label: "История операций", icon: Clock3 },
      { id: "settings", label: "Настройки", icon: Settings2 },
    ],
  },
];

export const sectionTitles: Readonly<Record<SectionId, string>> = {
  dashboard: "Обзор",
  vehicles: "Автомобили",
  diagnostics: "Диагностика",
  localization: "Локализация",
  "language-packages": "Языковые пакеты",
  adapters: "Адаптеры",
  "geely-research": "Исследование Geely",
  simulator: "Симулятор",
  history: "История операций",
  settings: "Настройки",
};

export const sectionDescription: Readonly<Record<SectionId, string>> = {
  dashboard: "Состояние локальной инженерной среды CARVION.",
  vehicles: "Каталог тестовых систем из локального хранилища.",
  diagnostics: "Проверка профилей адаптера без подключения к автомобилю.",
  localization: "Чтение переводимых строк из тестового Android XML.",
  "language-packages": "Подготовка к будущему workflow локализации.",
  adapters: "Доверенные локальные процессы, обнаруженные Go Core.",
  "geely-research": "Импорт локальных профилей и ресурсов; неизвестные конфигурации остаются unknown.",
  simulator: "Полный тестовый цикл CARVION без подключения к автомобилю.",
  history: "Журнал операций появится вместе с workflow проекта.",
  settings: "Сведения о текущем локальном запуске.",
};
