# Проверка переводов

`LocalizationValidationReport` содержит общий `valid`, locale, timestamp и упорядоченный список `issues`. Каждая проблема включает стабильный code, severity, resource key и сообщение. Severity `error` делает report невалидным.

## Проверки

- `missing_translation` — нет строки перевода для присутствующего переводимого ресурса.
- `empty_translation` — перевод пустой или состоит из пробелов.
- `duplicate_resource_key`, `duplicate_translation`, `duplicate_plural_category` — повторные ключи или plural categories.
- `unknown_translation_key` — перевод ссылается на ресурс вне проекта.
- `invalid_utf8` — source или target не является корректной UTF-8 строкой.
- `invalid_plural_category`, `plural_other_required` — неверная категория или отсутствует обязательный `other`.
- `invalid_format_arguments`, `format_arguments_mismatch` — malformed Android/Java format token или другой набор индексов и типов аргументов. Позиционные `%2$d`/`%1$s` можно переставлять.
- `source_changed_after_approval` — одобренный source SHA-256 больше не совпадает или перевод отмечен `needs_review`.
- `invalid_locale` — locale отсутствует или пустой; `unknown_translation_key` — перевод ссылается на ресурс вне проекта. Формат locale пока проверяется только на непустое значение.

Строки `translatable=false` и отсутствующие в текущем source snapshot ресурсы не требуют перевода. Report сохраняется в `localization_validation_runs`.

## Статусы

- `draft` — редактируемый перевод; пакет не может включать его.
- `approved` — пользовательское утверждение для конкретного source SHA-256.
- `needs_review` — исходник изменился или утверждённый перевод повторно редактировался.

При изменении исходника утверждение сбрасывается. После проверки перевод нужно утвердить повторно.
