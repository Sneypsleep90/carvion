# CARVION — описание проекта

## Назначение

CARVION — целевая **Desktop-платформа** для анализа и локализации программных ресурсов автомобильных мультимедийных систем через подключаемые адаптеры. Основное ядро пишется на Go; пользовательский интерфейс планируется на React и TypeScript внутри Wails 3. PostgreSQL указан как основная БД, JSON-RPC — как транспорт между ядром и отдельными процессами адаптеров.

## Текущее состояние

**Реализовано и проверено:** transport-independent Go Core (`domain`, `application`, `infrastructure`), composition root, repository ports, PostgreSQL read repository для исходного каталога, adapter bridge, Go HTTP API поверх Core, Adapter SDK v1, локальный Plugin Manager и `adapter-android-demo` с синтетическими данными. Есть unit/service tests, PostgreSQL repository integration test и HTTP tests.

**Не реализовано:** Desktop shell и Wails bindings, React UI, Hub-backed remote repository и Hub authentication, полноценные CRUD services, localization workflow, simulator, реальные автомобильные адаптеры, deployment и коммерческий Hub.

**Неизвестно / требует исследования:** локальная модель эксплуатации PostgreSQL в Desktop-продукте, достоверность и допустимость будущих источников данных, требования упаковки для целевых ОС.

## Цели

1. Сделать Desktop-приложение основным способом работы пользователя с CARVION.
2. Держать use cases в Go core отдельно от Wails, HTTP и UI.
3. Подключать адаптеры через версионированный SDK и контролируемые ядром источники данных.
4. Развивать анализ и локализацию только на подтверждённых данных и разрешённых методах.
5. Представлять неподтверждённую совместимость как `unknown`, не выдавая нехватку данных за поддержку.

## Текущая возможность демо

Демо принимает синтетическое JSON-наблюдение, сравнивает тестовый профиль по платформе, аппаратной ревизии и точной сборке ПО, а также извлекает переводимые строки из переданного Android `strings.xml`. Оно не подключается к автомобилю, не использует реальные образы прошивок и не заявляет поддержку Geely или другого производителя.

## Core и хранилище

`application.Service` вызывается из Go и не требует HTTP transport. Для разработки `composition.NewLocal` создаёт PostgreSQL connection pool, read-only каталог repository и существующий локальный Plugin Manager. Wails ещё не подключён. Repository ports позволяют добавить Hub-backed реализации; текущая удалённая реализация и механизм авторизации не существуют.
