# Прошивки и идентификаторы сборок

## Проверенные сведения

В публичных официальных источниках, перечисленных в [EVIDENCE.md](EVIDENCE.md), нет точной таблицы firmware build IDs, версий ПО и hardware revisions для Xingyue L / Monjaro 2022–2024. Описания платформ ОС и SoC из других годов не заменяют такие идентификаторы.

## Реализация CARVION

Versioned profile `carvion.geely-research/v1` предусматривает поля:

- `infotainment.os_name`, `os_version`;
- `hardware_identifier`, `hardware_revision`;
- `firmware_version`, `firmware_build_id`.

Эти поля принимаются как значения выбранного локального файла и отображаются `из файла / не проверено`. Их наличие не подтверждает подлинность или совместимость.

Проверенный каталог `internal/geelyresearch/compatibility_records.json` изначально пуст. Полное точное совпадение должно включать модель, год, рынок, комплектацию, infotainment platform, имя/версию ОС, hardware ID/revision и firmware version/build ID. Без любого поля, точной записи или проверки источника итог — `unknown`.

## Ограничения

CARVION не извлекает firmware image, не подключается к устройству, не загружает прошивку по сети, не записывает её и не предлагает неподтверждённый способ установки. Неизвестные build IDs нельзя подставлять или выводить из названия ОС/процессора.
