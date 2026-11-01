# Geely Research Adapter

## Состояние

**Реализован и проверен как исследовательский каркас.** CARVION Studio импортирует versioned JSON-профиль и отдельный Android XML или `carvion.localization/v1` JSON. Core передаёт данные процессу `adapter-geely-research` через локальный JSON-RPC, показывает происхождение полей, использует общий Localization Engine и экспортирует очищенный JSON-отчёт.

Это не адаптер для подключения к автомобилю и не подтверждение поддержки Geely. Реальные сведения по конкретным аппаратным ревизиям и сборкам 2022–2024 не подтверждены. Каталог точной совместимости пуст; текущий результат `unknown`.

## Поток

```text
Пользователь выбирает JSON/XML файлы
        │ typed Wails bindings
        ▼
desktop.ApplicationService → Core use cases
        ├── adapterbridge → Plugin Manager → adapter-geely-research
        └── LocalizationService → LocalizationRepository → PostgreSQL
```

Профиль принимается по схеме `carvion.geely-research/v1`. Ресурс отдельно анализируется тем же `internal/localization.Parse`, который использует Core. До сохранения проекта Core сравнивает ключи и исходный текст ответа адаптера с разбором Localization Engine. Создание проекта требует явного действия пользователя. Неизвестная совместимость не блокирует создание локализационного проекта и не означает, что ресурс подходит автомобилю.

## Версии и ограничения

- SDK API остаётся `v1`, SDK minor version — `1.1.0`; `vehicle_profile` — необязательное дополнение `CompatibilityRequest`.
- JSON-RPC adapter объявляет Detect, CheckCompatibility, InspectResources, XML и JSON форматы.
- Профиль ограничен 128 KiB; ресурс — 128 KiB. Desktop проверяет размер до чтения, а Core повторяет проверку. Обе границы укладываются в ограничение RPC frame.
- Compatibility rule должна совпасть по модели, model year, рынку, комплектации, platform, ОС, hardware ID/revision и firmware version/build ID. Неполное значение даёт `unknown`.
- Тестовые значения помечаются `TEST ONLY`; никаких реальных Geely build IDs в каталоге нет.
- Отдельной миграции и хранения исходного исследовательского файла нет. Сохранённый Localization project хранит выбранный пользователем ресурс обычным способом.

## Проверки

Профиль, exact matcher и privacy report покрыты Go unit tests; JSON-RPC process и resource inspection — process test; Wails facade — Go service tests; экран и импорт XML/JSON — Playwright E2E. PostgreSQL integration test для сохранения ресурса добавлен, но требует заданный `CARVION_TEST_DATABASE_URL`; фактический запуск отмечен в `docs/TESTING.md`.

Технические факты и источники перечислены в [EVIDENCE.md](EVIDENCE.md). Неизвестные поля — в [OPEN_QUESTIONS.md](OPEN_QUESTIONS.md).
