# CARVION — описание проекта

## Назначение

CARVION — **Desktop-платформа** для анализа и локализации программных ресурсов автомобильных мультимедийных систем через подключаемые адаптеры. Go Core отделён от транспорта; CARVION Studio использует Wails 3 и React/TypeScript, PostgreSQL — основной dev catalog store, JSON-RPC — транспорт к отдельным локальным процессам адаптеров.

## Текущее состояние

**Реализовано и проверено:** transport-independent Go Core (`domain`, `application`, `infrastructure`), composition root и repository ports; PostgreSQL catalog/localization repositories; HTTP API; Adapter SDK API v1 с manifest schema v2, локальный Plugin Manager и synthetic `adapter-android-demo`; Wails 3 app с React/TypeScript UI, typed direct bindings к Go Core и lifecycle-managed локальным runtime. Localization Engine импортирует Android XML и `carvion.localization/v1` JSON, редактирует/утверждает переводы, валидирует их, экспортирует CSV/ресурсы и собирает CARVION ZIP package. Актуальные test results см. в [TESTING.md](TESTING.md).

Simulator также реализован и проверен: он запускает локальные synthetic profiles через `adapter-android-demo`, а Go Core выполняет compatibility и тестовый localization/package workflow. Подробности — в [обзоре и результатах](simulator/OVERVIEW.md).

**Geely Research Adapter реализован как исследовательский каркас:** Desktop принимает пользовательский versioned JSON-профиль и отдельный XML/JSON ресурс, показывает значения как `из файла / не проверено`, анализирует ресурсы общим Localization parser, передаёт их в Localization Engine по явному действию и экспортирует очищенный диагностический отчёт. Поддержка Geely не заявлена: точные конфигурации 2022–2024 неизвестны, а каталог совместимости пуст. Источники и границы утверждений описаны в [docs/research/geely/](research/geely/OVERVIEW.md).

**Реализовано и проверено в синтетическом окружении:** Deployment Engine хранит jobs/events в PostgreSQL, проверяет CARVION ZIP integrity, выполняет preflight и запускает только synthetic provider. Desktop показывает проверки, результаты, прогресс и сохранённую историю. Production Mode безусловно блокируется в Go Core; PostgreSQL integration suite прогнан на отдельной временной БД. Проверок на автомобиле/физическом стенде не проводилось. Подробности и фактические ограничения — в [deployment docs](deployment/ARCHITECTURE.md).

**Реализовано частично:** Desktop экраны каталога и диагностики; отдельный Language Packages screen и Operation History screen остаются placeholders. История Deployment jobs доступна на экране Развёртывание. Сборка localization package доступна на экранах Localization, Simulator и перед Deployment preflight.

**Не реализовано:** Hub-backed remote repository и Hub authentication, реальные автомобильные адаптеры и production deployment/installation. CARVION Hub и коммерческая платформа запланированы.

**Неизвестно / требует исследования:** production модель хранения/авторизации Desktop (локальная PostgreSQL либо Hub), подписание и notarization приложения, достоверность и допустимость будущих источников данных.

## Цели

1. Сделать Desktop-приложение основным способом работы пользователя с CARVION.
2. Держать use cases в Go core отдельно от Wails, HTTP и UI.
3. Подключать адаптеры через версионированный SDK и контролируемые ядром источники данных.
4. Развивать анализ и локализацию только на подтверждённых данных и разрешённых методах.
5. Представлять неподтверждённую совместимость как `unknown`, не выдавая нехватку данных за поддержку.

## Текущая возможность демо

Демо принимает только профили из встроенного synthetic каталога, сравнивает их по платформе, аппаратной ревизии и точной тестовой сборке ПО, извлекает тестовые строки и может прогнать полный localization/package сценарий через Simulator. Localization Engine также принимает файлы пользователя напрямую. Проект не подключается к автомобилю, не использует реальные образы прошивок и не заявляет поддержку Geely или другого производителя.

## Core и хранилище

`application.Service` вызывается из Go и не требует HTTP transport. Wails facade в `desktop/` создаёт runtime через `composition.NewLocal`, передаёт Wails lifecycle context и закрывает pool при shutdown. Catalog и localization repository ports позволяют добавить Hub-backed реализации; удалённая реализация и auth отсутствуют. Студия сейчас требует локально настроенный PostgreSQL `DATABASE_URL` и не подключается к автомобилю.
