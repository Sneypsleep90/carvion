# Adapter SDK

## Версии

Импортируйте `github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1`. Текущий контракт объявляет `api_version: "v1"`, `sdk_version: "1.0.0"` и `manifest_version: 2`. Менеджер принимает только API v1 и SDK major version 1. Несовместимые или некорректные manifests не регистрируются.

Manifest хранится в `manifest.json` рядом с исполняемым файлом:

```json
{
  "manifest_version": 2,
  "api_version": "v1",
  "sdk_version": "1.0.0",
  "id": "adapter-example",
  "name": "Example Adapter",
  "version": "1.0.0",
  "executable": "bin/adapter-example",
  "capabilities": {
    "detect": true,
    "check_compatibility": true,
    "inspect_resources": false
  },
  "input_types": ["vehicle-observation-json", "compatibility-profile"],
  "resource_formats": []
}
```

`input_types` и `resource_formats` обязательны, даже если это пустые массивы. Тип входа должен соответствовать capability:

| Capability | Требуемый input type |
|---|---|
| `detect` | `vehicle-observation-json` |
| `check_compatibility` | `compatibility-profile` |
| `inspect_resources` | `resource-document` и не менее одного формата |

Формат ресурса задаётся стабильным slug, например `android-strings-xml`. Менеджер проверяет поля, типы, SemVer, дубликаты, соответствие capabilities и входов, а также совпадение `GetInfo` с файлом manifest.

## Типы и методы

Публичные типы SDK включают `AdapterManifest`, `AdapterCapabilities`, `VehicleIdentity`, `FirmwareInfo`, `HardwareInfo`, `CompatibilityResult` и request/result-типы операций. Интерфейс `Adapter` определяет:

- `GetInfo` — вернуть ту же версионированную manifest-информацию.
- `Detect` — интерпретировать JSON observation, подготовленный ядром.
- `CheckCompatibility` — сравнить переданный профиль с известными правилами.
- `InspectResources` — прочитать содержимое поддерживаемого документа.

Результаты определения и совместимости имеют строгое состояние: `supported`, `unsupported` или `unknown`. Используйте `unknown`, если точного подтверждения нет. Нельзя выводить поддержку из одного совпадения марки, имени или версии.

Метаданные manifest доступны ядру и экрану Adapters: adapter ID/version, SDK/API version, capabilities, input types, resource formats и последний runtime status (`available`/`unavailable`). Runtime status — результат последнего процесса, не фоновый health check.

См. [протокол процесса](PLUGIN_PROTOCOL.md) и [пример создания адаптера](CREATING_ADAPTER.md).
