# Ближайшие задачи: Step 6 — Geely Research Adapter

Step 5 CARVION Simulator завершён для локальных демонстрационных данных. Он проходит synthetic профили через JSON-RPC demo adapter, Go Core и Localization Engine; подробности и подтверждённые проверки — в [simulator/OVERVIEW.md](simulator/OVERVIEW.md) и [simulator/TEST_RESULTS.md](simulator/TEST_RESULTS.md). Полный порядок этапов — в [ROADMAP.md](ROADMAP.md).

## Следующий этап

1. Составить исследовательский план и перечень допустимых первичных источников по выбранной платформе.
2. Установить точные platform IDs, hardware revisions и firmware build identifiers только по подтверждённым данным.
3. Зафиксировать provenance, ограничения лицензий/доступа и неизвестные поля в `docs/research/`.
4. Не заявлять Geely support, пока идентификаторы и compatibility rules не подтверждены и не проверены.
5. Реализовывать Research Adapter только после фиксации evidence и безопасной read-only границы; неизвестные случаи должны оставаться `unknown`.

## Уже доступно в Desktop

- Состояние Go Core, локальной PostgreSQL и адаптеров; каталог и synthetic compatibility checks.
- Экран Adapters с версиями, capabilities и статусом.
- Localization projects: импорт Android XML/JSON, поиск и правка русских переводов, утверждение, повторный review после изменения источника, validation, CSV/resource export и package download.
- Simulator с демонстрационными профилями, compatibility exits, JSON-RPC ошибкой, отменой и проверкой скачиваемого CARVION ZIP.
- Desktop вызывает Go Core через Wails, не через HTTP.

## Оставшиеся продуктовые задачи

- Desktop runtime требует локальный `DATABASE_URL`; Hub-backed mode/auth и способ поставки PostgreSQL не решены.
- Android XML inline markup и CDATA внутри переводимой строки отклоняются; детали в [localization/FORMATS.md](localization/FORMATS.md).
- Language Packages и Operation History как отдельные экраны остаются placeholders.
- Simulator использует только встроенные synthetic fixtures, статические русские тестовые переводы и session-only event log; native Wails UI не покрыт автоматическим UI-тестом.
- Adapter process не является OS sandbox; signing/notarization, remote registry и real-car workflows не реализованы.
