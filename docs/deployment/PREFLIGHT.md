# Deployment preflight

Preflight фиксируется в job и содержит статус (`passed`, `blocked`, `failed`, `unknown`), признак обязательности и объяснение каждой проверки.

## Readiness audit

Аудит перед Step 7 подтвердил отсутствие production-ready платформы. Geely exact compatibility catalog пуст; прочие vehicle profiles — только synthetic. В проекте нет проверенного способа установки, точных production правил совместимости, документированного восстановления или стендовых результатов. Поэтому Production Mode безусловно заблокирован в Core.

## Проверки

| Check | Test Mode | Production Mode |
|---|---|---|
| `package-integrity` | Проверяет размер, ZIP entries, schema, non-OEM manifest, validation result и SHA-256 | Та же проверка перед сохранением blocked результата |
| `target-profile` | Требует профиль из встроенного synthetic каталога | Заблокировано: подтверждённой production платформы нет |
| `compatibility` | Требует точного synthetic supported tuple | `unknown`: production compatibility record отсутствует |
| `provider-capability` | Требует simulation-only provider и запрещает install capability | Блокируется |
| `installation-capability` | `blocked`/не требуется для simulation | Заблокировано: установочный метод не подтверждён |
| `result-verification` | Отдельная synthetic проверка package/result | Заблокировано: нет подтверждённых критериев проверки на платформе |
| `recovery-capability` | Synthetic capability check; это не rollback | Заблокировано: production recovery procedure отсутствует |
| `bench-evidence` | Не требуется для synthetic сценария, отражается как `unknown` | Заблокировано: стендовых данных нет |
| `production-policy` | Не применяется | Всегда `blocked` по политике Core |

Ни UI, ни конфигурационный файл не может переключить Core policy. Отрицательный, неполный или неизвестный профиль не повышается до `supported`.
