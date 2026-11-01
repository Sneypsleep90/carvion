# Deployment Engine — архитектура

## Состояние

**Реализовано и проверено unit/E2E проверками:** Core job lifecycle, package verifier, synthetic provider, PostgreSQL repository, Wails facade и Desktop workspace. PostgreSQL integration tests написаны и компилируются, но требуют отдельной test database; в текущей сессии они не выполнялись против сервера БД.

**Production deployment отсутствует и заблокирован политикой Core.** В коде `productionDeploymentEnabled = false`; настройки или UI не могут включить его.

## Границы компонентов

```text
React workspace
   │ typed Wails bindings + deployment:progress
   ▼
desktop.ApplicationService
   │ Start/Get/List/Cancel, worker lifecycle
   ▼
Core DeploymentService
   ├── LocalizationService → immutable saved package bytes
   ├── PackageVerifier → ZIP/schema/SHA-256 checks
   ├── DeploymentRepository → jobs + append-only events
   └── DeploymentProvider → synthetic-local-simulator only
                          └── no Install method/capability
```

Desktop вызывает Go bindings напрямую, без REST. Repository сохраняет job state и event sequence в PostgreSQL транзакционно. Wails progress event отправляется только после записи события; при пропущенном event UI перечитывает job из repository.

## Job lifecycle

```text
queued → preflighting → preflight-passed → simulating → verifying → simulated
                       ├→ blocked
                       ├→ failed
                       └→ cancelled
```

Неизвестная/неподдержанная synthetic profile блокируется до provider. Production job выполняет package integrity check, сохраняет blockers и заканчивается `blocked`. В Test Mode package integrity проверяется по ранее сохранённым bytes и точному SHA-256; затем требуется exact synthetic `supported` profile, simulation-only capability, отдельные synthetic result verification и recovery checks.

## Ограничения

- Нет install provider, install API, HTTP endpoint или SDK install RPC.
- Core передаёт provider только данные, нужные synthetic simulation; PostgreSQL credentials и DB handles не раскрываются.
- CARVION ZIP integrity не доказывает подпись, происхождение или OEM authenticity.
- Аудит append-only в рамках SQL repository; он не защищён от администратора PostgreSQL, владельца процесса или компрометации локальной машины.
- Проверки на автомобиле или физическом стенде не выполнялись.
