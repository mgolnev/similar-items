# План: версионированные генерации похожих товаров

- **Статус:** согласованное целевое направление
- **Дата:** 2026-07-24
- **Область:** `recomendationengine` и сценарии админки `recomendationenginegui`
- **Публичный контракт:** без изменений — сервис принимает артикул и возвращает упорядоченный список артикулов

## 1. Цель

Сохранить одну внешнюю ответственность RecommendationEngine:

```text
article + limit + exclude → ordered similar article IDs
```

Публичный API не должен:

- обогащать товары изображениями, ценами или региональными остатками;
- выполнять скоринг в пользовательском запросе;
- загружать исходный товар и его атрибуты из каталожной БД;
- иметь cold miss с тяжёлым расчётом;
- читать частично построенный результат.

Алгоритм выполняется отдельным builder-процессом после обновления фида или
публикации стратегии. API обслуживает только полностью построенную активную
генерацию.

## 2. Термины

- **Feed version** — полностью импортированная и проверенная версия каталога.
- **Strategy version** — опубликованная версия правил и весов.
- **Recommendation generation** — неизменяемый результат алгоритма для точной
  пары `(feed_version, strategy_version)`.
- **Active generation** — единственная генерация, доступная публичному API.
- **Builder** — фоновый batch-процесс, строящий новую генерацию.
- **Preview** — расчёт выбранных товаров по черновой стратегии без изменения
  active generation.

Recommendation generation — не TTL-кэш. У неё нет постепенного наполнения,
протухания и cold miss. Она строится целиком, проверяется и публикуется одной
атомарной операцией.

## 3. Архитектурные инварианты

1. Генерация всегда связана с конкретными `feed_version`, `strategy_version` и
   `algorithm_version`.
2. Публичный API читает только генерацию со статусом `active`.
3. Старая генерация обслуживает трафик, пока новая полностью не построена и не
   проверена.
4. Ошибка, timeout или отмена builder не меняют active generation.
5. Для одной целевой пары версий работает не более одного builder. В
   Kubernetes это отдельный Job/worker с PostgreSQL advisory lock, а не прогрев
   в каждой API-реплике.
6. Переключение active generation выполняется транзакционно.
7. API-запрос фиксирует generation ID в начале и до конца читает только её.
8. Сохраняются минимум две предыдущие генерации для быстрого rollback.
9. Preview и builder используют одно ядро скоринга. Отличаются только режим
   диагностики и версия стратегии.
10. Внешний ответ по-прежнему содержит только артикулы.

## 4. Целевой поток данных

```text
XML feed
  → streaming parse
  → normalize and validate
  → staging tables
  → feed_version ready
  → recommendation builder
  → generation validation
  → atomic active_generation switch
  → API lookup
  → BFF/catalog-cache enrichment
  → carousel
```

### 4.1. Импорт фида

1. Фид скачивается с allowlist URL, timeout и лимитом размера.
2. XML обрабатывается потоково, без `io.ReadAll` всего файла.
3. Признаки нормализуются один раз: секция, категории, цвет, аудитория,
   артикульное семейство, токены названия и attribute slots.
4. Данные загружаются bulk/COPY во staging.
5. Проверяются количество товаров, дубли, обязательные поля, секции,
   `publication_status` и `available`.
6. Создаётся новая feed version. Текущая рабочая версия не повреждается при
   ошибке.
7. Mini-feed получает отдельную семантику merge/preview и никогда не вызывает
   удаление всех SKU, отсутствующих в тестовом файле.

### 4.2. Builder

1. Один раз загружает подготовленный каталог нужной feed version.
2. Один раз компилирует strategy version в runtime-представление.
3. Делит товары по `women`, `men`, `kids`, `teens`; `all` не загружается
   отдельным повторным запросом.
4. Для каждого базового артикула рассчитывает независимые ранжированные списки
   уровней:
   - `strict`;
   - `parent`;
   - `branch`;
   - `fallback`.
5. Исключает базовый товар, запрещённое артикульное семейство и дубли модели.
6. Записывает generation батчами.
7. Вычисляет контрольные метрики и только после успешной проверки переводит
   generation в `ready`.
8. Отдельная транзакция переключает `active_generation`.

### 4.3. Публичный API

1. Нормализует входной article.
2. Читает active generation и готовые списки уровней.
3. Применяет пользовательские `exclude`.
4. Выбирает первый уровень, который после exclude способен заполнить
   запрошенный `limit`.
5. Возвращает первые `limit` ID.

В request path нет скоринга, сортировки, JSON-разбора стратегии и запросов к
таблице `products`.

## 5. Почему нужно хранить списки по уровням

Текущий `computeTopKIDs` запускает каскад с `limit=100`, после чего публичный
запрос обрезает результат до своего лимита.

Это меняет семантику. Например:

- strict содержит 60 товаров;
- materialization с `limit=100` переключается на parent;
- пользовательский запрос с `limit=12` при прямом расчёте остался бы на strict,
  но из кэша получает parent.

Пользовательские exclude также должны применяться до выбора fallback-уровня,
иначе после исключений strict может стать недостаточным, но API не перейдёт на
parent.

Поэтому новая generation хранит отдельные упорядоченные списки уровней, а API
выбирает уровень после применения exclude и фактического limit.

Размер materialized top-K задаётся отдельно от максимального API limit. Он
должен учитывать:

- максимальный публичный limit;
- пользовательские exclude;
- overscan, необходимый BFF перед фильтрацией региональной доступности.

Предварительно стоит проверить K=100 и K=200 на production-распределении
exclude и потерь после catalog-cache.

## 6. Предлагаемая модель хранения

Минимальная схема:

```text
recommendation_generations
  id
  feed_version
  strategy_version
  algorithm_version
  status                -- building | ready | active | failed | archived
  started_at
  completed_at
  base_count
  result_count
  build_statistics_json
  error

recommendation_lists
  generation_id
  article_id
  strict_ids[]
  parent_ids[]
  branch_ids[]
  fallback_ids[]

recommendation_runtime
  active_generation_id
  updated_at
```

Ключ `recommendation_lists(generation_id, article_id)` уникален и индексирован.
Для 11–12 тысяч базовых товаров PostgreSQL достаточно как источник правды.

После измерения production QPS доступны два serving-режима:

1. индексированный PostgreSQL lookup;
2. полная загрузка active generation в неизменяемый in-memory read model API.

Второй режим не является постепенно заполняемым кэшем: snapshot загружается
целиком и переключается целиком. Источником правды остаётся generation в БД.

## 7. Админка стратегии

### Draft и preview

- Save сохраняет только draft.
- Preview считает выбранные товары непосредственно алгоритмом по draft.
- Для сравнения старая и новая стратегия работают на одном каталоге и одном
  наборе базовых товаров.
- Showcase использует ограниченный worker pool и не блокирует production API.

### Build и publish

1. Стратегия валидируется и получает новую version.
2. Запускается builder.
3. Админка показывает generation ID, feed/strategy version, прогресс и ошибки.
4. Старая generation остаётся active.
5. После проверок новая generation активируется автоматически либо отдельным
   подтверждением — это нужно закрепить продуктовым решением.
6. UI явно различает `draft`, `building`, `ready`, `active`, `failed`.

Сохранение уже production-пресета не должно создавать состояние, в котором
документ изменился, а builder запустить невозможно.

## 8. Проверки перед активацией

Generation не активируется, если:

- обработаны не все допустимые базовые товары;
- присутствуют отсутствующие, неопубликованные или недоступные ID;
- товар рекомендует сам себя или запрещённую вариацию собственной модели;
- превышен допустимый процент пустых и недозаполненных списков;
- обнаружены межсекционные/межгендерные нарушения;
- feed, strategy или algorithm version не совпадают с целевыми;
- builder завершился частично.

Контрольные метрики:

- coverage базовых товаров;
- заполненность каждого уровня;
- overlap@K с предыдущей generation;
- доля полностью изменившихся top-N;
- category/gender leakage;
- количество удалённых и новых товаров;
- время и peak memory сборки;
- распределение score и fallback;
- доля результатов, отфильтрованных BFF/catalog-cache.

## 9. Повторный аудит алгоритма и узкие места

### 9.0. Baseline качества до изменений

Полный baseline снят 2026-07-24 на commit
`675c5db229f2598ed9a5504edde605aea612d057`, активной стратегии
`c0520bffa04f66392f4d84676ebdb31f456d34b98dc76e37d1611d51895e4fd6`
и 11 632 карточках. В `cmd`, `internal`, `go.mod` и `go.sum` на момент замера
не было локальных изменений.

Артефакты:

```text
artifacts/quality-baseline/baseline-675c5db-20260724/
  README.md
  summary.json
  catalog.csv.gz
  responses-limit-12.jsonl.gz
  responses-limit-40.jsonl.gz
  manual-review-sample.csv
```

Исходные показатели:

| Limit | Errors | Empty | Fill | Valid IDs | Same section | Same leaf | Same parent | Self/duplicates/same model |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 12 | 0% | 0% | 100% | 100% | 100% | 77.93% | 90.63% | 0 / 0 / 0 |
| 40 | 0% | 0% | 100% | 100% | 100% | 62.08% | 81.24% | 0 / 0 / 0 |

Для performance-only рефакторинга действует строгий gate: 100% exact ordered
match для каждого article и каждого captured limit. Осознанные изменения
бизнес-правил проходят отдельный quality diff и ручную проверку.

Воспроизводимые инструменты:

```bash
python3 scripts/capture-quality-baseline.py \
  --output-dir artifacts/quality-baseline/<new-baseline>

python3 scripts/compare-quality-baselines.py \
  --before artifacts/quality-baseline/baseline-675c5db-20260724 \
  --after artifacts/quality-baseline/<new-baseline> \
  --mode exact
```

### 9.1. Зафиксированный baseline

На Apple M1, Go benchmark с 1 500 кандидатами и активной стратегией:

```text
17.6–19.5 ms/op
~2.77 MB/op
7 506 allocs/op
```

Локальный полный warm 11 632 товаров с семью workers занимал около 12 минут
22 секунд.

CPU/heap profile показывает:

- `scoreAttributeRules` — около 44% cumulative CPU;
- создание `RuleScores map[string]int` — около 42% alloc space;
- `RankSimilarCandidates` и полный `scoredPair` — около 34% alloc space;
- в горячем пути остаются `strings.ToLower`, Unicode-normalization, string-map
  lookup и создание category leaf map;
- полный массив кандидатов сортируется, хотя generation нужны только
  ограниченные top-K списки.

### 9.2. Корректность до оптимизации

Перед ускорением необходимо тестами зафиксировать и осознанно исправить:

1. `conflict_penalty` цвета сейчас отбрасывается условием `score <= 0`.
2. `MissingPolicy` хранится, но не участвует в вычислении.
3. `popular_in_section` не использует popularity; fallback фактически
   детерминирован score/SKU.
4. XML `available` заменяется на `true`.
5. Каскад materialized top-100 меняет поведение меньших limit и exclude.
6. Правило удаления всего каталога при mini-feed.

Изменение этих пунктов влияет на бизнес-выдачу и должно сопровождаться golden
diff и продуктовым подтверждением.

### 9.3. Рефакторинг горячего пути

#### A. Разделить serving score и explain

Создать два режима:

- `scorePairFast` возвращает только `score`, `baseWithoutTitle` и tier;
- `scorePairExplain` строит `ScoreBreakdown`, `RuleScores`, notes и details.

Builder использует только fast-режим. Карты диагностических баллов не
создаются для миллионов пар.

#### B. Компилировать стратегию один раз

Ввести `CompiledStrategy`:

- правила уже отфильтрованы по enabled и section;
- scope нормализован;
- match type преобразован в enum/function kind;
- веса и caps находятся в компактных полях;
- rule IDs заменены индексами;
- словари title/color собраны один раз;
- strategy hash вычислен один раз.

Внутри пары не должно быть `TrimSpace`, `ToLower` и прохода по scope strings.

#### C. Компактные подготовленные признаки товара

Отделить транспортный `ProductRow` от `PreparedProduct` builder:

- числовой product/model/family ID;
- category IDs и нормализованные leaf sets;
- предвычисленные teen section/class;
- dense rule feature slice по индексам compiled rules вместо
  `map[string]ruleFeat`;
- token IDs/bitsets или отсортированные integer slices вместо строковых maps;
- готовые article prefix и group key.

Цель fast scorer — ноль аллокаций на пару.

#### D. Убрать полную сортировку

Во время одного прохода поддерживать bounded top-K для strict, parent, branch
и fallback. Сложность становится `O(N log K)` вместо `O(N log N)`.

Дедуп модели должен сохранять лучшего кандидата до попадания в итоговый heap.
Для этого предпочтительны integer model IDs и переиспользуемый generation
marker array, а не string map на каждую базовую карточку.

#### E. Загружать каталог один раз

Builder получает одну согласованную catalog snapshot и создаёт секционные
срезы в памяти. Не нужно пять раз:

- выполнять запросы к products;
- объединять group attributes;
- парсить attributes JSON;
- строить title/rule features.

#### F. Записывать результаты батчами

Использовать `COPY`/batch insert в `recommendation_lists`. Нельзя выполнять
отдельную транзакцию на каждый article.

### 9.4. Реализованные performance-only этапы

2026-07-24 выполнены два этапа без изменения бизнес-формулы:

- production/builder scoring отделён от подробного admin explain;
- fast path больше не создаёт `ScoreBreakdown`, `RuleScores` и notes;
- признаки rules, title, catalog leaf и teen category вычисляются один раз на
  товар, а не внутри каждой пары;
- synonym groups названия также вычисляются один раз на товар; исправлен
  benchmark, который раньше не загружал синонимы активной стратегии;
- правила хранятся плотным срезом по индексам; applicability по секции и
  match-kind компилируются до ранжирования;
- production scorer работает с указателями и не копирует `ProductRow` на
  каждом вызове;
- category/model/family/article-prefix признаки интернированы в `uint32` внутри
  одного prepared catalog; строковый fallback сохранён для admin/cold path;
- каталог для warm загружается и подготавливается один раз, секционные пулы
  строятся в памяти;
- полная сортировка заменена точным per-tier bounded top-K после выбора лучшей
  цветомодификации каждой модели.

Результаты на том же Apple M1:

| Измерение | Baseline | После fast scorer | Текущий результат |
|---|---:|---:|---:|
| Rank 1 500 кандидатов | 17.6–19.5 ms/op | 2.31–3.20 ms/op | 1.80–1.91 ms/op |
| Allocated bytes, rank 1 500 | ~2.77 MB/op | 49 256 B/op | 49 256 B/op |
| Allocations, rank 1 500 | 7 506 allocs/op | 4 allocs/op | 4 allocs/op |
| Production select 11 632 | — | 20.8–22.6 ms/op¹ | 13.8–15.5 ms/op |
| Allocated bytes, select 11 632 | — | 3.65 MB/op¹ | 0.41 MB/op |
| Полный warm 11 632, 7 workers | 12m22s | 1m08s | 33s end-to-end / 25s scoring |

¹ Контрольный полный `RankSimilarCandidates + PickSimilarFromRanked` на
текущем numeric scorer, чтобы отдельно измерить эффект bounded top-K.

От первоначального baseline полный warm ускорился примерно в **22.5 раза**
end-to-end. Pair scorer не создаёт объектов; 11 аллокаций короткого benchmark
top-K относятся к bounded heaps, model map и итоговому ID slice.

Quality gate:

```text
limits: 12 and 40
articles per limit: 11 632
exact ordered output: 100%
position match: 100%
errors / empty / unknown / self / duplicates / same-model: 0
```

Артефакты результата и exact comparison:

```text
artifacts/quality-baseline/refactor-numeric-topk-final-20260724/
  README.md
  summary.json
  responses-limit-12.jsonl.gz
  responses-limit-40.jsonl.gz
  comparison-exact.json
```

Exact gate сравнил 23 264 ответа (11 632 артикула для limit 12 и 40):
ordered output, позиции и aggregate quality metrics совпали на 100%.

### 9.5. Реализованный serving fast path

2026-07-24 после обновления локального feed до 11 668 eligible cards выполнен
отдельный performance-only этап публичного API:

- `LoadProductBySKU` сначала выполняет primary-key lookup по `sku`; только
  отсутствующий SKU переходит к индексируемым `group_id`/`sales_sku` aliases и
  затем к legacy prefix fallback;
- cache-hit загружает только identity товара (`sku`, aliases, section, gender),
  не объединяет attributes и не готовит title/rule features;
- hash активной стратегии считается при её применении, а не на каждом запросе;
- batch из 50 артикулов разрешает identities одним SQL и напрямую читает
  прогретые ID; последовательные 50 вызовов `FindSimilar` удалены;
- cold miss и admin explain сохранили полный ProductRow/scoring path.

До изменения запрос с `OR + LIKE + ORDER BY sku LIMIT 1` для позднего артикула
выполнял scan `products_pkey`, отбрасывал 11 667 строк и занимал около 31 ms.
Прямой primary-key lookup на том же PostgreSQL занимал около 0.19 ms.

Результат полного black-box capture с восемью concurrent batch workers:

| Limit | Метрика batch latency | До | После |
|---:|---|---:|---:|
| 12 | mean | 811 ms | 10.3 ms |
| 12 | p50 | 796 ms | 7 ms |
| 12 | p95 | 1 335 ms | 31 ms |
| 40 | mean | 794 ms | 9.7 ms |
| 40 | p50 | 787 ms | 9 ms |
| 40 | p95 | 1 222 ms | 19 ms |

Контролируемый повтор одного batch из 50 артикулов показал p50 около 4.1 ms
без конкуренции и 9.4 ms при concurrency 8. Hot single lookup для раннего и
позднего SKU имеет p50 около 1.1–1.2 ms.

Для корректного quality gate предыдущий commit `5dfcc8c` был запущен отдельно
на той же обновлённой БД. Exact gate сравнил 23 336 ответа (11 668 артикулов,
limits 12 и 40):

```text
exact ordered output: 100%
position match: 100%
errors / missing / changed: 0
```

Артефакты:

```text
artifacts/quality-baseline/serving-before-5dfcc8c-feed-20260724/
artifacts/quality-baseline/serving-fastpath-final-20260724/
  comparison-exact.json
```

## 10. Этапы реализации

### Этап 0. Контракт и baseline

- Зафиксировать production QPS и SLA.
- Зафиксировать max limit, exclude budget и BFF overscan.
- Добавить golden dataset и snapshot текущей выдачи.
- Добавить benchmark полного generation build на production-подобном фиде.

### Этап 1. Семантика алгоритма

- Отдельно согласовать popularity, zero-score fallback, negative penalties и
  missing policy.
- Исправить подтверждённые ошибки под golden-diff контролем.
- Добавить тест каскада для разных limit/exclude.

### Этап 2. Fast scorer

- Разделить fast/explain.
- Ввести `CompiledStrategy` и `PreparedProduct`.
- Добиться отсутствия аллокаций на пару.
- Ввести per-tier bounded top-K.
- Подтвердить эквивалентность fast и explain score тестами.

### Этап 3. Generation storage и builder

- Добавить version/status tables.
- Реализовать singleton builder и batch persistence.
- Добавить проверки, rollback и retention.
- Оставить текущий serving path без изменений за feature flag.

### Этап 4. Lookup API

- Реализовать чтение active generation.
- Корректно применять exclude, tier cascade и limit.
- Убрать products query, strategy hash и cold calculation из нового path.
- Добавить version headers/structured logs для диагностики.

### Этап 5. Admin lifecycle

- Draft → preview → build → validate → activate.
- Прогресс и ошибки builder.
- Сравнение old/new generation.
- Возможность rollback.

### Этап 6. Shadow и rollout

- На части запросов сравнивать текущий и generation path без изменения ответа.
- Сравнить latency, errors и exact ID output.
- Переключить feature flag после прохождения acceptance criteria.
- Удалять текущий `idCache`/warm path только после стабильного production
  периода.

## 11. Тестовая стратегия

- Unit: scoring rules, tier cascade, exclude, model dedupe, deterministic ties.
- Equivalence: fast scorer == explain scorer по итоговому score.
- Golden: old/new top-N diff на фиксированном feed/strategy.
- Integration PostgreSQL: staging → feed version → generation → activation →
  API.
- Concurrency: параллельные feed/strategy updates и singleton builder.
- Failure injection: timeout, cancelled build, database error, pod restart.
- Multi-replica: все API-поды сходятся на одном active generation.
- Load: lookup API и полный builder отдельно.
- Race tests для lifecycle и in-memory active snapshot.

## 12. Предварительные acceptance criteria

- Публичный API не выполняет scoring и не читает `products`.
- Нет частично активной generation и cold calculation.
- Ошибка builder не влияет на текущую выдачу.
- Fast scorer создаёт 0 аллокаций на пару в benchmark.
- Generation build минимум в 3 раза быстрее текущего baseline; окончательный
  целевой бюджет уточняется после production resource/QPS данных.
- Lookup API укладывается в согласованный p95/p99 под production-подобной
  нагрузкой.
- Все изменения top-N объясняются утверждёнными semantic fixes.
- Rollback generation выполняется одной административной операцией.

## 13. Решения, которые ещё нужно закрепить

Они не блокируют первые benchmark/refactoring этапы, но блокируют production
активацию:

1. Целевые QPS, p95 и p99.
2. Максимальный limit, реальный exclude budget и требуемый BFF overscan.
3. Источник и формула popularity.
4. Допустима ли пустая карусель вместо zero-score fallback.
5. Автоматическая или ручная активация generation после build.
6. Допустимое время между поступлением фида и активацией новой generation.
7. Сколько предыдущих generations хранить.
8. Нужна ли глобально синхронная активация на всех репликах или достаточно
   атомарности каждого запроса и быстрой сходимости.
