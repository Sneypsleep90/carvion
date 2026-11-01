# Безопасность Deployment Engine

## Гарантии текущей реализации

- Production Mode отключён compile-time policy в Go Core и завершается сохранённой blocked job.
- Единственный зарегистрированный provider — `synthetic-local-simulator`; в capability contract `CanInstall=false`.
- Provider interface не содержит `Install`; SDK/JSON-RPC не расширялся операциями записи.
- Core проверяет exact synthetic profile, совместимость, hash пакета и отдельные result/recovery capabilities до запуска симуляции.
- Jobs и append-only события сохраняются в PostgreSQL. UI получает их через typed Wails bindings.
- Никакой прямой доступ к машине, файловой системе, сети или PostgreSQL provider не предоставлен.

## Границы обещаний

SHA-256 подтверждает только совпадение проверяемых байтов с указанным digest. Он не удостоверяет издателя или подпись. CARVION language package имеет `oem_firmware=false` и не является OEM образом.

Тестовый provider подтверждает результат только внутри своей синтетической модели. Он не подтверждает безопасность, пригодность, совместимость или восстановление реального автомобиля. На физических системах испытаний не было.

## Вне объёма

Не реализованы firmware write, установочные команды, Secure Boot/signature/auth bypass, hardware access и production provider. Для пересмотра блокировки потребуются допустимые проверяемые доказательства платформы, installation method, compatibility, result verification, recovery, bench validation и отдельное архитектурное решение. Эта документация не является одобрением таких действий.
