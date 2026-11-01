# Создание адаптера

## Минимальная реализация

1. Создайте Go module или command, импортирующий `github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1`.
2. Реализуйте интерфейс `sdk.Adapter`.
3. Верните из `GetInfo` manifest schema v2 с теми же значениями, что в установленном `manifest.json`; текущая SDK package version — `1.1.0`, API version — `v1`.
4. Запустите `sdk.Serve(ctx, os.Stdin, os.Stdout, adapter)` в `main`; логи пишите только через stderr.
5. Соберите executable внутрь каталога адаптера и укажите относительный путь в manifest.
6. Укажите только действительно реализованные capabilities, входные типы и форматы.
7. Поместите пакет в доверенный локальный `ADAPTERS_DIR`, затем перезапустите CARVION.

Для нового адаптера сначала скопируйте структуру [demo manifest](../../adapters/adapter-android-demo/manifest.json), заменив ID, имя, версию и capabilities. Не копируйте synthetic profile как реальные данные.

## Локальная проверка

```sh
go test ./pkg/adaptersdk/v1 ./internal/pluginmanager ./internal/adapters/androiddemo ./internal/adapters/geelyresearch
make build-adapter
make desktop-dev
```

Проверки Manager должны покрывать валидный/невалидный manifest, SDK/API mismatch, declared inputs/formats, RPC errors, malformed response, process exit и timeout. Compatibility tests должны включать точный профиль и неизвестные build/vehicle. В Desktop проверяйте только capability-совместимые варианты; отсутствие подходящего адаптера должно блокировать действие.

Не логируйте содержимое ресурсов и observations без необходимости. Не добавляйте автоматический сетевой поиск или download executable.
