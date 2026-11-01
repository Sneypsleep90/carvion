# Восстановление и отказ

## Текущий статус

Реальной процедуры восстановления нет. Production readiness check сохраняет `recovery-capability: blocked`; до записи на оборудование дело не доходит. В проекте нет production install provider, поэтому нечего автоматически откатывать.

Synthetic provider может сообщить `available=true, simulated=true` только как результат тестовой capability check. Это не backup, rollback, recovery image, проверка состояния автомобиля или рекомендация по ремонту.

## Отказы job

- Неверный/повреждённый package завершается `failed` на preflight; provider не запускается.
- Неизвестный либо неподдержанный target завершается `blocked` до provider.
- Provider/RPC error, timeout и cancellation сохраняют terminal state и event в PostgreSQL.
- Wails facade отменяет worker при shutdown и ограниченно ожидает его завершения.
- Progress сначала сохраняется, затем публикуется. История восстанавливается через Core Get/List bindings.

Результат `failed` или `cancelled` не означает, что автомобиль был изменён. В рамках текущего provider изменения оборудования невозможны.

## Что нужно для production recovery

Неизвестно и требует отдельного исследования: подтверждённая платформенная процедура восстановления, допустимые backup/restore механизмы, условия безопасной остановки, критерии результата и стендовые испытания. Пока эти данные отсутствуют, production capability должна оставаться заблокированной.
