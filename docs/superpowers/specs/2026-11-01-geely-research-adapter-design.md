# CARVION Geely Research Adapter — Design

**Статус:** пользователь одобрил общее направление 2026-11-01. Этот документ требует review перед подготовкой плана реализации.

## Цель

Добавить в CARVION Studio исследовательский режим для локальных диагностических файлов Xingyue L / Monjaro 2022–2024. Это исследовательский workflow, а не заявление о совместимости. При отсутствии точной проверенной записи результат остаётся `unknown`.

## Доказательная база и ограничения

На дату исследования 2026-11-01 доступны официальные материалы Geely, но найденные сведения не задают проверенную матрицу точных OS/build IDs, аппаратных ревизий, годов, комплектаций и рынков за 2022–2024:

| Источник | Что подтверждает | Границы вывода |
|---|---|---|
| [Geely: запуск Xingyue L](https://global.geely.com/en/news/2021/geely-launches-suv-xingyuel), опубликовано 2021-07-20; официальный источник, высокая уверенность в описании запуска | Для запуска Xingyue L заявлялись Milky Way OS и многоэкранная система | Не задаёт точные сборки или аппаратные ревизии 2022–2024 |
| [Geely: спецификации Xingyue L, PDF](https://dm30webimages.geely.com/GeelyPromotion/KX11/2021/pzb1029.pdf), официальный материал 2021 года по URL; дата публикации внутри документа не установлена | Перечислены разные варианты комплектации; в таблице конфигураций указана Snapdragon 8155 | Не доказывает соответствие каждого года/рынка/комплектации 2022–2024 и build ID |
| [Geely: таблица характеристик Xingyue L 2025, PDF](https://global.geely.com/-/media/project/web-portal/parallel-car/xingyuel-01/geely-xingyue-l-dongfangyao-specification-table.pdf), документ обозначен как 2025; получен 2026-11-01 | Для перечисленных вариантов 2025 года указаны Galaxy OS 2.0 и Snapdragon 8155 | Не является источником для 2022–2024 |
| [Geely: текущая страница Monjaro](https://global.geely.com/en/models/monjaro), проверена 2026-11-01; официальный актуальный источник | Текущая страница указывает Geely ONE OS 2.0 / Snapdragon 8155 и предупреждает о региональных различиях | Не устанавливает историческую конфигурацию или совместимость 2022–2024 |
| [Geely: публикация о продажах за 2025 год](https://global.geely.com/en/news/2026/geely-auto-sales-2025), опубликована в 2026; официальный источник | В публикации Xingyue L соотнесён с названием Monjaro для зарубежных рынков | Не подтверждает одинаковое ПО, оборудование или сборки между рынками |

Эти материалы можно показывать как ограниченные публичные сведения с указанным scope. Ни один из них не является compatibility record. Исходный каталог точных правил поддержки пуст. В документации отдельно помечаются факт, вывод и неизвестное; даты публикации и получения не смешиваются.

## Подходы

1. **Исследовательский отдельный процесс — выбран.** Сохраняет Plugin Manager/SDK границу, принимает только переданные Core данные и не имеет доступа к базе или машине.
2. **Импортёр только в Core.** Не удовлетворяет цели подключаемого Geely adapter.
3. **Матрица поддержки из общих рекламных спецификаций.** Отклонён: найденные источники не подтверждают точные соответствия для целевых годов.

## Архитектура

```text
React Geely Research Mode
        │ typed Wails bindings; user-selected file contents
        ▼
desktop.ApplicationService
        │ direct call; context, validation and size limits
        ▼
application.Service / Geely Research use cases
   ├── AdapterGateway → Plugin Manager → adapter-geely-research (JSON-RPC)
   └── existing LocalizationService → LocalizationRepository → PostgreSQL
```

- Добавить общий пакет `internal/geelyresearch` для versioned profile, evidence references, strict validation, exact matching и report schema.
- Добавить процесс `cmd/adapter-geely-research`, реализацию в `internal/adapters/geelyresearch` и manifest под `adapters/adapter-geely-research/`.
- Адаптер объявляет только file observation, compatibility check и resource inspection. Не запускает устройство и не собирает данные сам.
- Core читает файлы через выбранный пользователем Desktop `File` input, проверяет содержимое и размер. В JSON-RPC передаются значения/содержимое, basename и format; исходный filesystem path адаптеру не передаётся.
- Для разбора Android XML и `carvion.localization/v1` JSON переиспользуется `internal/localization.Parse`; не создаётся второй parser. Тот же содержимый файл может быть импортирован через существующий Localization Engine в проект PostgreSQL.
- Resource inspection — анализ файла, а не подтверждение применимости ресурса к конкретной машине. Неизвестная совместимость не блокирует локализационный импорт; UI и отчёт сохраняют явное предупреждение `unknown`.
- Новый Wails экран называется «Исследование Geely». HTTP routes и frontend REST вызовы не добавляются.
- Новые таблицы и хранение raw diagnostic files не нужны. Текущий профиль и результаты исследовательского сеанса живут в Desktop session; Localization Engine сохраняет импортированный ресурс обычным проектом.

### Profile v1

Профиль JSON имеет строгую версионированную схему `carvion.geely-research/v1`. Он хранит только сообщённые/прочитанные из файла наблюдения:

- идентификатор схемы и собственный ID профиля;
- производитель, маркетинговое имя, model year, market/region и trim;
- infotainment platform;
- имя и версия ОС;
- аппаратный идентификатор/ревизия;
- версия ПО и firmware build ID;
- ссылки на источники с названием, издателем, URL, датой публикации, датой получения и областью применимости.

Отсутствующее значение кодируется как неизвестное, а не заполняется предположением. Пользовательский JSON не может сам пометить значение как проверенный факт: его значения отображаются как `из файла / не проверено`. Ссылки, переданные файлом, также не становятся проверенными источниками автоматически. Схема не принимает VIN, credentials или произвольные дополнительные поля; неизвестные поля отклоняются.

### Exact compatibility

- Совместимость сравнивается целиком: модель, год, рынок/регион, комплектация, infotainment platform, OS name/version, hardware ID/revision, firmware version и build ID.
- `supported` возможен только при точном полном совпадении с отдельной curated-записью, у которой есть проверяемый источник и review metadata.
- `unsupported` возможен только при точной curated-записи об исключении для этой комбинации. Несовпадение или неизвестное значение само по себе не означает `unsupported`.
- Неполный профиль, неизвестный build или отсутствие записи дают `unknown` с объяснением.
- Начальный каталог compatibility records для Geely пуст: ни один целевой автомобиль/профиль пока не будет объявлен `supported`.
- SDK `CompatibilityRequest` сегодня не передаёт year/market/trim/OS. Для полного сравнения потребуется минимальное additive расширение SDK v1: необязательный JSON `vehicle_profile` в compatibility request и соответствующий новый SDK minor version. Поле отсутствует в обычных запросах и передаётся только Geely Research workflow. Нужно сохранить текущий размер RPC envelope; импортируемый профиль ограничить так, чтобы запрос вместе с framing оставался в `MaxRPCMessageBytes`.

### Resource analysis and Localization Engine

- Поддерживаемые файлы: Android `strings.xml` (`<string>`, `<string-array>`, `<plurals>`) и JSON только по существующей схеме `carvion.localization/v1`.
- SDK adapter `InspectResources` возвращает ключи и исходные строки через общий parser. Исходная структура, metadata, plurals и массивы сохраняются Localization Engine, который остаётся источником данных проекта.
- Валидация формата, UTF-8, размера, duplicate keys и неполного JSON/XML выполняется до сохранения. Пользователь подтверждает создание Localization project; текущая локаль по умолчанию `ru-RU`.
- Ограничения существующего parser, включая отказ на nested XML markup/CDATA, сообщаются явно.

### Desktop и диагностический отчёт

- Экран позволяет загрузить versioned JSON профиль, отдельно выбрать XML/JSON ресурс, увидеть все наблюдения и состояние происхождения, запустить profile classification/compatibility check, увидеть список ресурсов и открыть исходник в Localization Engine.
- Видимые состояния: `из файла / не проверено`, `подтверждено источником`, `unknown`, `supported`, `unsupported`, `ошибка`. Публичные материалы Geely показываются с годом и областью применимости; исходные материалы 2021/2025/2026 не выдаются за 2022–2024 evidence.
- Export создаёт JSON diagnostic report с версией схемы, нормализованными разрешёнными полями, их происхождением, временем отчёта, совместимостью, списком ресурсов, hash входного файла/ресурса, source references и ограничениями.
- Отчёт не включает исходное содержимое ресурсов, полный локальный путь, VIN, секреты или содержимое firmware image. Файл профиля не допускает такие поля.
- UI явно обозначает, что это импортированный файловый профиль, а не соединение/обнаружение реального автомобиля.

## Ошибки и защита

- Только локальные файлы, выбранные пользователем; никакой загрузки по URL и сетевого получения плагинов.
- Ограниченные размеры профиля, ресурсов и отчёта; строгий JSON decoder; проверка UTF-8; basename-only filename; очищенные сообщения ошибок.
- Адаптер получает JSON-RPC content только для текущего вызова и не получает `DATABASE_URL`, DB handles, filesystem path, device handle или вход к CAN/OBD/ADB.
- Не добавлять операции установки/записи firmware, обхода защиты, shell command из профиля или обращения к реальному автомобилю.
- Plugin Manager запускает доверенный локальный executable, но сам отдельный процесс не является OS sandbox; docs должны сохранить это ограничение.

## Тестирование дизайна

- Unit: schema strictness, missing/unknown observations, provenance preservation, exact match по каждому ключу, `unknown` при каждом отсутствующем поле, запрет unsupported inference, report sanitization.
- Adapter process: GetInfo/manifest, Detect profile from file as unverified identity (not vehicle connection), CheckCompatibility exact-vs-unknown, XML/JSON inspection using shared parser, malformed input and RPC deadline/errors.
- Core/Wails: context propagation, adapter selection by fixed ID/capabilities, import-to-localization handoff, report export, errors and no filesystem path leakage.
- Frontend E2E: JSON/XML selection, field provenance display, unknown badge, Localization project import and report download; no scenario may claim real vehicle connection.
- PostgreSQL integration: verify a selected resource project persists through the existing LocalizationRepository; no schema migration is planned.
- `make desktop-test`, `make lint`, `make build`, `make desktop-build` and `git diff --check`; document only commands actually run.

## Вне scope

- Утверждения о поддержке конкретных Monjaro/Xingyue L сборок без проверяемых источников.
- CAN/OBD/ADB/USB/Bluetooth discovery, загрузка данных по сети и подключение к автомобилю.
- Firmware installation, flashing, reverse engineering, bypass защиты или OEM packaging.
- Raw firmware ingestion, архивирование диагностических файлов, VIN и аккаунтные данные.
- Новые HTTP endpoints, remote Hub repositories, административный каталог правил и новые PostgreSQL migrations.

## Открытые вопросы, которые исследование не разрешает

- Какие фактические комбинации ГУ, рынка, trim, OS, hardware revision и firmware build встречались на Xingyue L / Monjaro в 2022–2024.
- Есть ли документированный официальный формат диагностического файла для этих конфигураций.
- Есть ли OEM service/user manuals и разрешённые источники идентификаторов для каждого рынка.
- Какие варианты Android resource files реально встречаются, включая XML dialects и JSON schemas.

До подтверждения этих сведений работа остаётся файловым исследовательским каркасом; compatibility records не добавляются.
