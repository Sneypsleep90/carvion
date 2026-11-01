# Поддерживаемые форматы локализации

Все входные данные должны быть UTF-8. Невалидные байты отклоняются; автоматическая замена символов не выполняется.

## Android `strings.xml`

Поддерживаемый корневой элемент — `<resources>`. Engine извлекает:

- `<string name="…">text</string>` с атрибутами `translatable` и `formatted`;
- элементы `<string-array name="…"><item>…</item></string-array>` с ключами `name[index]`;
- элементы `<plurals name="…"><item quantity="…">…</item></plurals>` с ключами `name#category`.

Сохраняются исходные байты файла, комментарии вне переводимого текста, порядок, атрибуты, атрибуты контейнера и метаданные. При экспорте заменяются только текстовые диапазоны переводимых элементов, XML escaping применяется к новому тексту. Пустые self-closing ресурсы разворачиваются в парные теги при экспорте перевода. Повторные ключи, namespaced корень или `string`/`item` elements, неверные boolean attributes и malformed XML завершаются ошибкой. Неизвестные Android resource elements остаются в исходном документе без изменений и не добавляются в переводимый каталог. Nested markup, CDATA, comments и processing instructions внутри поддерживаемой переводимой строки пока не поддерживаются и отклоняются при импорте.

Допустимые plural categories: `zero`, `one`, `two`, `few`, `many`, `other`. Для каждой группы требуется `other`.

## JSON `carvion.localization/v1`

Документ обязан объявлять schema и массив ресурсов. Неизвестные поля отклоняются.

```json
{
  "schema": "carvion.localization/v1",
  "metadata": { "source": "synthetic example" },
  "resources": [
    {
      "key": "home.title",
      "kind": "string",
      "text": "Home",
      "translatable": true,
      "formatted": false,
      "metadata": { "comment": "Synthetic screen title" }
    },
    {
      "key": "track_count#other",
      "groupKey": "track_count",
      "kind": "plural-item",
      "pluralCategory": "other",
      "text": "%d tracks"
    }
  ]
}
```

`kind` принимает `string`, `string-array-item` или `plural-item`. Для array items передаются `groupKey` и `itemIndex`; для plural items обязательны `groupKey` и `pluralCategory`. При отсутствии `translatable`/`formatted` используются `true`. Поля `metadata` документа и ресурса сохраняются при export. Экспорт выдаёт JSON с отступами и завершающим переводом строки.

## CSV translations

Импорт и экспорт используют фиксированный RFC 4180 header:

```csv
resource_key,source_text,target_text,locale
home.title,Home,Главная,ru-RU
```

Порядок и названия колонок обязательны. Запятые, кавычки и переносы строк кодируются стандартным CSV writer. CSV import принимает только текущие `resource_key`, точный `source_text` и locale проекта; импортированные записи получают статус `draft`. Дубликаты ключей в одной локали и невалидный UTF-8 отклоняются.
