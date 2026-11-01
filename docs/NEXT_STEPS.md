# Ближайшие задачи: Step 4 — Localization Engine

Step 3 завершён: SDK API v1 с manifest schema v2, Plugin Manager, process JSON-RPC и тестовый адаптер интегрированы с Desktop. Следующий этап — workflow локализации. Roadmap: [ROADMAP.md](ROADMAP.md).

## План работ Step 4

1. **Workflow анализа.** Развить XML inspection до проекта локализации с устойчивыми resource IDs и исходным текстом.
2. **Редактирование и review.** Определить модель переводов, статусы проверки и журнал изменений без скрытого изменения исходных файлов.
3. **Экспорт.** Спроектировать проверяемый формат языкового пакета и валидацию до записи результата.
4. **Хранение.** Определить локальный project store и границы будущей Hub synchronization.
5. **Тесты.** Покрыть parsing, duplicate IDs, invalid input, escaped XML, редактирование и export round-trip.

## Уже доступно в Desktop

- Обзор состояния Go Core, локальной PostgreSQL и обнаруженных адаптеров.
- Список и сведения о synthetic demo adapter.
- Read-only каталог систем из PostgreSQL.
- Ручной compatibility check и synthetic Android `strings.xml` inspection через Go binding facade.
- Выбор установленного адаптера по заявленной capability и формату ресурса; runtime status показывает последний вызов.
- Восемь разделов навигации; языковые пакеты, история и часть настроек пока являются объясняющими placeholder views.

## Ограничения и вопросы

- Desktop runtime сейчас требует настроенный локальный `DATABASE_URL`; Hub-backed mode и auth не реализованы.
- Нужно решить, поставляется ли PostgreSQL локально с Desktop либо Desktop MVP работает только через Hub. Решение должно сохранять отсутствие административных DB credentials в клиенте.
- Текущие Desktop tests проверяют Go service facade и React E2E на браузерном тестовом backend; packaged app запускается вручную для smoke проверок.
- Localization workflow, сохранение проектов, translation editing, экспорт и operation history относятся к Step 4.
