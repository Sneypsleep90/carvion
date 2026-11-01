# CARVION — описание проекта

## Назначение

CARVION — **Desktop-платформа** для анализа и локализации программных ресурсов автомобильных мультимедийных систем через подключаемые адаптеры. Go Core отделён от транспорта; CARVION Studio использует Wails 3 и React/TypeScript, PostgreSQL — основной dev catalog store, JSON-RPC — транспорт к отдельным локальным процессам адаптеров.

## Текущее состояние

**Реализовано и проверено:** transport-independent Go Core (`domain`, `application`, `infrastructure`), composition root и repository ports; PostgreSQL read repository; HTTP API; Adapter SDK API v1 с manifest schema v2, локальный Plugin Manager и synthetic `adapter-android-demo`; Wails 3 app с React/TypeScript UI, typed direct bindings к Go Core и lifecycle-managed локальным runtime. Desktop показывает возможности/статус адаптеров и позволяет выбрать adapter ID для synthetic calls. Актуальные test results см. в [TESTING.md](TESTING.md).

**Реализовано частично:** Desktop экраны каталога/диагностики и read-only XML inspection; отдельные UI разделы пока placeholders.

**Не реализовано:** Hub-backed remote repository и Hub authentication, полноценные CRUD services, localization editing/export workflow, simulator, реальные автомобильные адаптеры, deployment и коммерческий Hub.

**Неизвестно / требует исследования:** production модель хранения/авторизации Desktop (локальная PostgreSQL либо Hub), подписание и notarization приложения, достоверность и допустимость будущих источников данных.

## Цели

1. Сделать Desktop-приложение основным способом работы пользователя с CARVION.
2. Держать use cases в Go core отдельно от Wails, HTTP и UI.
3. Подключать адаптеры через версионированный SDK и контролируемые ядром источники данных.
4. Развивать анализ и локализацию только на подтверждённых данных и разрешённых методах.
5. Представлять неподтверждённую совместимость как `unknown`, не выдавая нехватку данных за поддержку.

## Текущая возможность демо

Демо принимает синтетическое JSON-наблюдение, сравнивает тестовый профиль по платформе, аппаратной ревизии и точной сборке ПО, а также извлекает переводимые строки из переданного Android `strings.xml`. Оно не подключается к автомобилю, не использует реальные образы прошивок и не заявляет поддержку Geely или другого производителя.

## Core и хранилище

`application.Service` вызывается из Go и не требует HTTP transport. Wails facade в `desktop/` создаёт runtime через `composition.NewLocal`, передаёт Wails lifecycle context и закрывает pool при shutdown. Repository ports позволяют добавить Hub-backed реализации; удалённая реализация и auth отсутствуют. Студия сейчас требует локально настроенный PostgreSQL `DATABASE_URL` и не подключается к автомобилю.
