# CARVION Language Packages

Package Builder создаёт детерминированный ZIP только после успешной валидации и утверждения каждой присутствующей переводимой строки для текущей версии исходника.

## Состав

```text
manifest.json
validation.json
resources/<source filename>
checksums.sha256
```

- `manifest.json` объявляет `schema: carvion.language-package/v1`, `package_type: carvion-localization`, locale, формат, путь ресурса и source SHA-256.
- `oem_firmware` всегда `false`. Пакет явно идентифицирован как CARVION localization artifact.
- `validation.json` содержит результаты проверок; volatile `checkedAt` опускается для воспроизводимой сборки.
- `checksums.sha256` содержит SHA-256 каждого предыдущего файла в сортированном порядке. Сам файл checksum не хеширует.
- ZIP entries сортируются, используют фиксированную дату и Store compression. Для одинакового входа байты ZIP и SHA-256 совпадают.

Пакет сохраняется в таблицу `language_packages` вместе с ZIP bytes, размером, SHA-256 и ссылкой на validation run. UI получает artifact через Wails binding и сохраняет его как `.carvion.zip`.

Пакет предназначен для обмена/проверки локализации. Он не является OEM firmware, не прошивается и не устанавливается на автомобиль.

Deployment Engine проверяет ограниченный набор ZIP entries, schema, validation report, entry checksums и SHA-256 artifact. Это проверяет целостность относительно переданного digest, но не издателя/подпись; при несоответствии provider не запускается.
