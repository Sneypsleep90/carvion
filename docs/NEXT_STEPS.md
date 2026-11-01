# Ближайшие задачи

Step 6 добавил Geely Research Mode для выбранных локальных файлов. Это исследовательский каркас; точные конфигурации Xingyue L / Monjaro 2022–2024 неизвестны, Geely support не заявлен. См. [исследовательские документы](research/geely/OVERVIEW.md) и [общий порядок этапов](ROADMAP.md).

## Следующий этап — Step 7: Deployment Engine (пока только проектирование)

1. Согласовать разрешённый scope операций и подтвердить применимые официальные процедуры/источники до реализации.
2. Спроектировать проверку целостности, provenance, явное подтверждение пользователя, cancellation и безопасный отказ.
3. Разделить package validation и подтверждение реального результата; локальный тест не должен изображаться подтверждением на автомобиле.
4. Если нет проверенной и разрешённой процедуры для конкретной конфигурации, deployment должен оставаться выключенным/неизвестным.
5. Не реализовывать обход защиты, несанкционированную запись или неподтверждённую установку.

## Доступно сейчас

- CARVION Studio напрямую вызывает Go Core через Wails bindings.
- Adapters показывает локальные процессы; `adapter-android-demo` и `adapter-geely-research` работают через JSON-RPC.
- Simulator проходит synthetic сценарии без настоящего автомобиля.
- Localization импортирует Android XML и versioned JSON, редактирует/проверяет русский перевод и строит CARVION language package.
- Geely Research Mode принимает JSON-профиль и отдельный XML/JSON ресурс, показывает публичные источники с scope, классифицирует без точного правила как `unknown`, импортирует ресурс по явному действию и экспортирует отчёт без local path/raw resource content.

## Оставшиеся ограничения

- Geely hardware revisions и firmware build IDs для целевых лет не подтверждены; exact catalog пуст.
- PostgreSQL integration test нового Geely import требует отдельный `CARVION_TEST_DATABASE_URL` и в текущем запуске не выполнялся.
- Desktop dev режим требует локальную PostgreSQL; Hub repository и Hub auth не реализованы.
- Android XML nested inline markup/CDATA пока отклоняется.
- Language Packages и Operation History остаются отдельными placeholder экранами.
- OS sandbox, plugin signing, native UI automation, macOS notarization и distribution update ещё не реализованы.
