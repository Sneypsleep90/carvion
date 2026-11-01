# Источники и границы утверждений

**Дата получения материалов:** 2026-11-01. Уверенность относится только к тому ограниченному факту, который прямо указан; ни один источник ниже не подтверждает совместимость CARVION.

| ID | Источник | Издатель | Дата публикации | Получено | Подтверждает | Не подтверждает | Уверенность |
|---|---|---|---|---|---|---|---|
| `geely-xingyue-l-launch-2021` | [Geely launches Xingyue L](https://global.geely.com/en/news/2021/geely-launches-suv-xingyuel) | Geely Auto | 2021-07-20 | 2026-11-01 | В описании запуска названы Milky Way OS и многоэкранный кокпит Xingyue L | Точные OS/build ID и аппаратные ревизии 2022–2024 | Высокая для содержания публикации, отсутствует для совместимости |
| `geely-xingyue-l-launch-specifications-2021` | [Xingyue L launch specifications (PDF)](https://dm30webimages.geely.com/GeelyPromotion/KX11/2021/pzb1029.pdf) | Geely Auto | Внутри документа не установлена; URL указывает путь 2021 | 2026-11-01 | Несколько вариантов начальной комплектации; таблица указывает Snapdragon 8155 | Применимость к каждой модели/рынку 2022–2024 или любой build ID | Высокая для указанной таблицы, отсутствует для совместимости |
| `geely-xingyue-l-specifications-2025` | [2025 Xingyue L specification table (PDF)](https://global.geely.com/-/media/project/web-portal/parallel-car/xingyuel-01/geely-xingyue-l-dongfangyao-specification-table.pdf) | Geely Auto | 2025 (обозначение документа) | 2026-11-01 | Для перечисленных вариантов 2025 года указаны Galaxy OS 2.0 и Snapdragon 8155 | Конфигурации 2022–2024 или взаимозаменяемость вариантов | Высокая для указанной таблицы, отсутствует для совместимости |
| `geely-monjaro-current-product-page` | [Geely Monjaro product page](https://global.geely.com/en/models/monjaro) | Geely Auto | Текущая страница, дата публикации не указана | 2026-11-01 | На момент получения указаны Geely ONE OS 2.0, Snapdragon 8155 и региональные различия | Исторические сочетания 2022–2024 и точные builds | Высокая для текущего содержимого страницы, отсутствует для исторической совместимости |
| `geely-xingyue-l-monjaro-naming-2026` | [Geely 2025 sales and 2026 targets](https://global.geely.com/en/news/2026/geely-auto-sales-2025) | Geely Auto | 2026 | 2026-11-01 | Публикация соотносит Xingyue L с названием Monjaro для зарубежных рынков | Идентичность аппаратуры/ПО между рынками и годами | Высокая для формулировки публикации, отсутствует для совместимости |

Эти записи доступны в коде через `internal/geelyresearch.PublicEvidence()` и показаны в Desktop с датой получения и scope. Пользовательские ссылки в imported profile не входят в этот каталог и не повышают статус проверки.

## Вывод по доказательной базе

В исходном `compatibility_records.json` нет правил. В репозитории не зафиксированы реальные Geely build IDs, hardware revisions или `supported`/`unsupported` комбинации. Поэтому текущий compatibility status для любого Geely профиля — `unknown`. Общие продуктовые сведения не допускаются как доказательство точного правила совместимости.
