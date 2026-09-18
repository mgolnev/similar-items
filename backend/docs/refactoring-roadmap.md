# Refactoring Roadmap — GJ Similar

Технический долг, выявленный при аудите сервиса (июнь 2026). Сервис рабочий: `go build`/`go vet`/`go test` зелёные, покрытие тестами приличное. Рефакторинг — точечный, по очагам, **без изменения внешнего поведения** (контракт публичного API и алгоритм похожих не меняются).

Принцип: каждый шаг завершается зелёными `go build ./... && go vet ./... && go test ./...`; где затронут Postgres-кодпас — прогоняем интеграционные тесты против реального Postgres (`make pg-up`, `INTEGRATION_DATABASE_URL=...`).

## Статус

| # | Очаг | Подзадача | Статус |
|---|------|-----------|--------|
| 1 | Поддержка двух СУБД | 1b. Надёжный SQL-rewriter + тесты трансляции | ✅ |
| 1 | Поддержка двух СУБД | 1a. Анти-дрейф схемы: канонический манифест + тест согласованности | ✅ |
| 2 | God-файлы | 2a. Разбить `product_repository.go` (1929→1560) | ✅ |
| 2 | God-файлы | 2b. Разбить `data_upload_service.go` (1262→642) | ✅ |
| 3 | Наследие automerch | Аудит/срез неиспользуемой поверхности | ✅ |

## Итоги (что сделано)

- **1b:** `rewriteBooleanSQL` переписан на регэкспы (устойчив к пробелам/алиасам), добавлен `connection_rewrite_test.go` с покрытием всех веток трансляции.
- **1a:** `schema_consistency_test.go` — unit-проверка канонических колонок products (SQLite) + интеграционный `TestSchemaParity_SQLiteVsPostgres` (требует `INTEGRATION_DATABASE_URL`), сверяющий наборы таблиц/колонок. Задокументированные исключения: `online_sales/online_purchases/offline_sales/score` (денормализация Postgres). Добавлен `postgres_integration_test.go` (репозиторный поток на реальном Postgres). Verified на свежем Postgres (`make pg-up`).
- **2a:** доменная логика вынесена из `product_repository.go` в `product_grouping.go`, `product_sizes.go`, `product_dates.go` (тот же пакет — поведение неизменно).
- **2b:** `data_upload_service.go` → `data_upload_sales.go` (sales enrichment) + `data_upload_scores.go` (score recalculation).
- **3:** срезан подтверждённо-мёртвый код (см. ниже).

## Очаг №1 — поддержка двух СУБД (SQLite + Postgres)

**Решение:** SQLite **сохраняем** (на нём гоняются unit-тесты без контейнера + локальный режим). Цель — убрать хрупкость, а не выпиливать вторую СУБД.

### 1b. SQL-rewriter (`internal/database/connection.go`)
Проблема: `rewriteBooleanSQL` делает литеральные строковые замены (`"p.available = 1"` → `TRUE`), завязанные на точное форматирование. Любой новый запрос с другим пробелом молча не транслируется.

- Заменить литеральные замены boolean на регэксп по известным boolean-колонкам (`available`, `is_active`, `enabled`), устойчивый к пробелам и алиасам.
- Полноценные unit-тесты `rewritePostgresSQL` на все ветки: placeholders (в т.ч. `?` внутри строковых литералов), boolean, `INSERT OR REPLACE`→`ON CONFLICT`, `INSERT OR IGNORE`, `json_extract`/`json_array_length`, `datetime('now')`, удаление `INDEXED BY`.

### 1a. Анти-дрейф схемы
Проблема: схема описана трижды по-разному — SQLite (реестр из 33 инкрементальных миграций), Postgres (единый снапшот `createPostgresSchema`), плюс расхождение моделей (в Postgres `products` денормализует `online_sales/score`, в SQLite они в `product_metrics`/`scores`).

- Ввести **канонический манифест** (таблица → ожидаемые колонки) как единый источник правды в Go.
- Тест: поднять in-memory SQLite, прогнать миграции, интроспектировать `PRAGMA table_info`, сверить с манифестом; распарсить колонки из `createPostgresSchema` и сверить с манифестом. Любой будущий дрейф → красный тест.
- Известные расхождения (денормализация Postgres, мёртвая константа `queries.GetProducts`) задокументировать; полную унификацию модели данных делать отдельно с прогоном против реального Postgres.

## Очаг №2 — God-файлы

### 2a. `internal/repository/product_repository.go` (1929 строк, ~45 функций)
Мешает data-access и доменную логику. Вынести в отдельные пакеты/файлы чистые функции без обращения к БД:
- группировка товаров (`mergeProductIntoGroup`, `GroupProductsForScoring`, `MerchGroupKey`, `normalizeGroupKey`);
- размеры (`sortProductSizes`, `applySizeStockoutFromSizes`, `*SizeOutOfStockPercent*`);
- даты (`parseSaleDateValue`, `pickEarliestDate`, `saleStartDateFromAttributesJSON`).

Репозиторий оставляет только методы доступа к данным. Поведение неизменно — переносим как есть, тесты переезжают рядом.

### 2b. `internal/services/data_upload_service.go` (1262 строки)
Слишком много ответственностей. Выделить:
- **sales enrichment** (online/offline/sale-start из Excel) → отдельный сервис;
- **score recalculation** (`StartScoreRecalculation`, `RecalculateAllScores`, `ExplainScore`, статусы) → отдельный сервис.

`DataUploadService` остаётся фасадом фид-пайплайна + scheduler.

## Очаг №3 — наследие automerch

README заявляет, что search/outfits/weights/categories-админка удалены, но в схеме и репозиториях ещё живут: таблицы `search_*`, `product_outfit_*`, репозитории `Weights`/`Category`/`Order`, пакет `clientcatalog`, мёртвая константа `queries.GetProducts`.

### Карта аудита (по grep живого кода, без миграций/тестов)

| Поверхность | Вердикт | Действие |
|---|---|---|
| `queries.GetProducts` (константа) | dead (0 ссылок) | удалена |
| `search_zero_results`, `search_query_log`, `search_query_clicks` | dead (0 ссылок) | удалены из SQLite-миграций + Postgres-схемы |
| `product_outfit_blocks`, `product_outfit_rules` | dead (0 ссылок) | удалены (вместе с `migrations_outfits.go`) |
| FTS: `products.search_tsv` + `RunPostgresSearchMigrations` (`migrations_postgres_search.go`) | dead (0 ссылок) | удалены |
| `products.online_sales/online_purchases/offline_sales/score` (только Postgres) | dead: фид их не пишет, читаются из таблиц `online_sales/offline_sales/scores` через JOIN; в SQLite уже выпилены миграцией `products_new` | срезаны из Postgres-схемы |
| таблица `product_metrics` (обе СУБД) | dead: только `DELETE`, ни одного `INSERT`/`SELECT` | удалена (схемы + раннер + DELETE-вызовы) |
| запросы `GetProductBySKU`, `UpdateProduct`, `CreateProduct`, `ExportAllProducts`, `ExportCategoryProducts`, `GetGeneralStats`, `GetCategoryStats`, `ClearProductMetrics` | dead (0 ссылок); единственные читатели/писатели срезанных колонок | удалены |
| `category_order`, `global_product_order`, `category_scores` | **живые** (ручная сортировка в админке) | оставлены |
| `Weights`/`Category`/`Order`-репозитории, `clientcatalog` | **живые** (через `container` + репозитории) | оставлены |

Проверка среза: миграции на чистом Postgres зелёные, `TestSchemaParity_SQLiteVsPostgres` подтверждает согласованность схем, репозиторный поток работает. На уже существующих БД старые таблицы остаются пустыми и безвредными (DROP не выполняется — просто перестаём их создавать на новых развёртываниях).
