# Адаптеры CARVION

## Состояние

**Реализовано и проверено:** SDK API v1, manifest schema v2, локальная discovery в доверенном каталоге, process-per-call JSON-RPC 2.0, синтетический `adapter-android-demo`, файловый `adapter-geely-research` и Desktop экраны адаптеров/Geely Research. Geely adapter анализирует только выбранные пользователем профили и ресурсы; поддержку автомобиля не заявляет.

**Реализовано, но не является production sandbox:** менеджер ограничивает время, размер RPC frame, число параллельных вызовов и окружение процесса. Процесс всё ещё работает с правами пользователя CARVION.

**Запланировано:** signed distribution, remote Hub registry, OS sandbox, синхронизация PostgreSQL adapter tables и разрешённые источники реального сбора данных. Для Geely exact catalog пуст; без полного подтверждённого профиля результат остаётся `unknown`.

## Документы

- [SDK и manifest](SDK.md)
- [JSON-RPC protocol](PLUGIN_PROTOCOL.md)
- [Создание адаптера](CREATING_ADAPTER.md)
- [Границы безопасности](SECURITY.md)

## Код

- `pkg/adaptersdk/v1` — API types, manifest validation и JSON-RPC server.
- `internal/pluginmanager` — trusted local discovery, регистрация, запуск, timeout и проверка ответов.
- `internal/core/infrastructure/adapterbridge` — соединение Manager с application Core.
- `internal/adapters/androiddemo` — synthetic demo implementation и fixture data.
- `internal/geelyresearch`, `internal/adapters/geelyresearch` — строгая profile schema, ограниченные evidence, exact compatibility и file-only JSON-RPC adapter.
- `adapters/adapter-android-demo/manifest.json` — manifest для локального пакета.
- `adapters/adapter-geely-research/manifest.json` — manifest исследовательского процесса.
- `desktop/frontend/src/features/adapters-page.tsx` — список, версии, capabilities и runtime status.
- `desktop/frontend/src/features/geely-research-page.tsx` — импорт профиля/ресурсов, evidence, explicit Localization import и diagnostic report.

Подробный локальный запуск описан в [RUN.md](../../RUN.md), Desktop — в [DESKTOP.md](../DESKTOP.md).
