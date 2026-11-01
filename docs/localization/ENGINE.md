# Localization Engine

Localization Engine хранит переводимые ресурсы проекта, проверяет переводы и создаёт отдельный CARVION language package. Функции Desktop вызываются напрямую через Wails bindings в `desktop.ApplicationService` и Go Core. HTTP routes для локализации не добавлены.

## Компоненты

- `internal/localization` — чистые Go парсеры Android XML и JSON, CSV import/export, validator, resource exporter и deterministic ZIP builder.
- `internal/core/application.LocalizationService` — импорт проекта, перевод, утверждение, повторный импорт исходника, проверка, экспорт и package build.
- `internal/core/application.LocalizationRepository` — storage port; Core не зависит от pgx.
- `internal/core/infrastructure/postgres.LocalizationRepository` — транзакционное хранилище на pgx без ORM.
- `desktop.ApplicationService` и `desktop/frontend/src/features/localization-page.tsx` — Wails boundary и интерфейс.

## Жизненный цикл

1. Пользователь выбирает XML или JSON файл. UI декодирует его как UTF-8 с ошибкой на невалидных байтах и передаёт содержимое Core.
2. Parser проверяет формат, извлекает resource keys, source text и метаданные. Проект и исходный файл сохраняются одной PostgreSQL транзакцией; исходник хранится как неизменяемый снимок до повторного импорта.
3. Перевод сохраняется как `draft`. Пользователь отдельно утверждает строку — `approved`; запись содержит SHA-256 исходной строки.
4. При повторном импорте изменённые и удалённые строки переводятся в `needs_review`, сохраняется запись в history с hash исходника, для которого ранее было утверждение. В активной записи approval hash очищается. Неизменившиеся утверждения сохраняются.
5. Validator создаёт report и сохраняет его. Exporter может выгрузить локализованный XML/JSON или CSV.
6. Package Builder требует валидный report и утверждённый перевод каждой присутствующей переводимой строки.

## Хранение

Миграция `db/migrations/00003_localization_engine.sql` создаёт проекты, ресурсы, переводы, append-only историю, validation runs и artifacts language packages. Foreign keys связывают перевод с ресурсом, утверждённый перевод — с точным source SHA-256, а пакет — с validation run того же проекта. Запросы выполняются через параметризованный pgx SQL.

## Ограничения текущей версии

- Одна локаль у проекта; текущий интерфейс по умолчанию использует `ru-RU`.
- Locale проверяется на непустое значение; синтаксис и каноникализация BCP 47 пока не реализованы.
- Android XML поддерживает `<string>`, `<string-array>` и `<plurals>`. Вложенная разметка внутри переводимого элемента и CDATA отклоняются, чтобы exporter не потерял форматирование.
- JSON принимает только объявленную схему `carvion.localization/v1`.
- Источник импортируется локальным пользователем; автоматического сканирования каталогов или сетевой загрузки нет.
- PostgreSQL обязательна в текущей Desktop composition. Hub-backed/local-file repository не реализованы.
- Package — архив ресурсов CARVION. Он не является прошивкой и не предназначен для прямой установки в автомобиль.
