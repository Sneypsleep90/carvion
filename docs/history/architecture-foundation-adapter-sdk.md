# Исторический архитектурный снимок: Foundation и Adapter SDK

> Этот документ сохраняет обзор первоначальной реализации до Step 1 Desktop-First Core Refactoring. Он не описывает текущие package boundaries. Актуальная архитектура находится в [`../ARCHITECTURE.md`](../ARCHITECTURE.md).

## Исходная схема

```text
HTTP client
    │
    ▼
CARVION API (Go + chi)
    ├── /healthz, /readyz
    ├── /api/v1/adapters
    ├── /api/v1/compatibility/check
    ├── PostgreSQL pool (pgxpool)
    └── Plugin Manager
          └── JSON-RPC 2.0 stdin/stdout
                 └── adapter-android-demo (synthetic fixtures)
```

В первоначальном варианте `cmd/carvion-api` одновременно загружал config, подключал PostgreSQL, обнаруживал локальные adapters и создавал HTTP server. SDK и Plugin Manager уже работали отдельно от HTTP, но application service layer ещё отсутствовал.

## Реализованные компоненты исходного этапа

- PostgreSQL schema через Goose migrations: Foundation tables и Step 2 adapter/firmware/compatibility tables.
- `pkg/adaptersdk/v1` с `GetInfo`, `Detect`, `CheckCompatibility`, `InspectResources` и compatibility states.
- Локальный Plugin Manager с versioned manifest, executable path checks, process timeout и ограниченным JSON-RPC framing.
- Synthetic Android demo для тестовой JSON identity, точного тестового firmware profile и Android `strings.xml`.
- HTTP endpoints для списка адаптеров и compatibility check.

## Границы доверия

Адаптер не получает PostgreSQL credentials, pool, file path входного ресурса или device handle. Core передаёт данные payload через RPC. Процесс адаптера не является OS sandbox и наследует системные права пользователя; запускать можно только owner-trusted executables.

Тестовый профиль не является реальными данными автомобиля и не заявляет поддержку Geely или иного производителя. Firmware installation и подключение к реальным автомобилям отсутствуют.
