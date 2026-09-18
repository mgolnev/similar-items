# ADR 0001 — Выделение сервиса «Похожие» из automerch

- **Статус:** Принято
- **Дата:** 2026-06-01
- **Контекст:** монолит `automerch` (Go/Fiber + React) совмещал каталог, фид, скоринг, поиск, образы (outfits) и похожие товары в одном процессе.

## Решение

Выделить функциональность «Похожие товары» в самостоятельный сервис `similar-items`
(`platform/similar/`) со своим репозиторием, БД, деплоем и админкой.

Сервис включает:

1. **Загрузку фида по расписанию** (`FEED_SYNC_INTERVAL`, дефолт `1h`) — собственный пайплайн импорта XML, нормализации discovery-полей и пересчёта скоринга.
2. **Алгоритм похожих** (`internal/discovery/similar`) — rule-based ранжирование.
3. **Админку стратегии** — `/api/admin/similar-strategy*` + React-страницы `SimilarStrategy*`.
4. **Публичный API** — `/api/v1/client/products/:id/similar` и batch-вариант.

## Что перенесено из automerch (максимальное переиспользование)

Перенесён и адаптирован в модуль `similar-items`:

- Каталоговый/скоринговый слой: `catalog`, `clientcatalog`, `queries`, `utils` (XML-парсер, scoring).
- Доступ к данным: `repository` (product, scores, sales, sale_start_dates, weights, category, feed_categories, sync, similar, runtime_settings).
- Миграции БД: `database` целиком (products + discovery-колонки, scores, runtime_settings; неиспользуемые таблицы оставлены безвредно для совместимости реестра миграций).
- Сервисы: `data_upload_service` (фид + scheduler), `strategy_service`.
- Стратегия: `discovery/strategy` (+ `section`).
- HTTP: `admin_strategy`, `data_upload`, similar-часть `client_handler`.
- Frontend: `SimilarStrategy` + вкладки (Preview/Colors/Rules/FeatureCatalog), `strategyApi`, типы, `normalizeStrategy`, `featureCatalogHints`; добавлены витрина `SimilarShowcase` и страница `Feed`.

## Что удалено (не относится к «Похожим»)

`discovery/search`, `discovery/searchstrategy`, `outfits`, `catalogfacet`, клиентский каталог
(`client_catalog_*`, навигация витрины), `search_*`/`outfit_*` репозитории и их сервисы,
админки поиска/категорий/навигации/весов, утилита `sqlite-to-postgres`, `catalog-perf`,
legacy `web/` UI. Для этой функциональности остаётся монолит `automerch`.

## Последствия

- **Плюсы:** независимый деплой и масштабирование рекомендаций; чистая граница ответственности; меньше связности.
- **Минусы / открытые вопросы:**
  - Сервис поддерживает собственный пайплайн фида и каталоговый read-model; в будущем возможен общий read-model/CDC.
  - Интеграция `search → similar` в монолите (адаптер `SimilarExpander`) при необходимости заменяется HTTP-вызовом к этому сервису.

## Границы

- Сервис не импортирует код automerch и не использует его БД.
- `internal/discovery/similar` не зависит от поиска/каталога-витрины/outfits.
