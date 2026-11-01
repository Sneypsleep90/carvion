# CARVION — запуск

## Требования

- macOS 13+ на Apple Silicon
- Go 1.27.3, Node.js/npm, Docker Compose, Xcode Command Line Tools
- Wails CLI v3.0.0: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`

## Первый запуск

Из корня репозитория:

```sh
cp .env.example .env
make db-up
make migrate-up
make desktop-dev
```

Запустится CARVION Studio с локальной PostgreSQL и тестовым адаптером. На экране Simulator доступны только встроенные synthetic profiles; Desktop вызывает Go Core через Wails bindings и не использует локальный REST API.

## Production-сборка

`make desktop-build` создаёт `.app`; `make desktop-run` собирает и запускает её.

Приложение: `desktop/bin/carvionstudio.app`.

## API (опционально)

В отдельном терминале:

```sh
make build
make dev
```

API доступен на `http://localhost:8080`.

## Тесты и остановка

```sh
make desktop-test
make lint
make build
# остановить Desktop/API через Ctrl+C
make db-down
```
