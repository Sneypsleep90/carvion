# Сценарии CARVION Simulator

Все identity, hardware, firmware и XML значения в таблице — synthetic fixtures из `internal/simulator/testdata/`.

| Profile ID | Ожидаемый compatibility | Поведение |
|---|---|---|
| `demo-android-platform-a` | `supported` | Полный цикл: Detect → exact Compatibility → Android XML inspection → Localization → validation → CARVION ZIP → проверка SHA-256, checksums и non-OEM manifest. |
| `demo-android-platform-b` | `supported` | Второй профиль платформы с отдельной synthetic сборкой; адаптер должен распознать его точно. |
| `unknown-firmware` | `unknown` | Identity известен только как демонстрационный; незарегистрированная сборка останавливается до импорта ресурсов. |
| `incompatible-hardware` | `unsupported` | Build совпадает с Platform A, но аппаратная ревизия отличается; пакет не создаётся. |
| `simulated-communication-failure` | `unknown` ожидаемо; сценарий фактически `failed` | Demo adapter возвращает контролируемую JSON-RPC ошибку на синтетическом observation; UI показывает ошибку и event log. |

Профиль `unknown-car`, отсутствующий в локальном каталоге, отклоняется Core как invalid input до вызова адаптера. Unknown firmware и неизвестный автомобиль — разные случаи: первый представляет известную synthetic identity с неучтённой тестовой сборкой.

## Отказ и границы

- Любой compatibility результат `unknown` или `unsupported` останавливает workflow до `InspectResources` и создания Localization project.
- Ошибка процесса/JSON-RPC или отмена контекста создаёт `failed` результат с последним событием; последующие шаги не запускаются.
- Неуспешный package hash, ZIP, resource, validation report, checksum или OEM marker запрещает завершить сценарий как успешный.
- Сценарии не запускают установку, запись, автомобильное соединение, ADB, CAN или OBD.

Event log отображается на экране Simulator только для текущего результата и не является общей persistent Operation History.
