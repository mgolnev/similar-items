# «Комплекты»: план и контракт MVP

## Цель

Добавить второй независимый алгоритм рекомендаций: комплектность вещь + вещь по явным ссылкам фида. Алгоритм не меняет формулу «Похожих товаров» и имеет собственные настройки, API, preview и generation-cache.

## Поток данных

1. XML-фид сохраняет параметр `Сочетается с товаром` в `products.attributes`.
2. Outfit builder загружает опубликованные доступные карточки с изображениями.
3. Варианты одной карточки объединяются, ссылки из всех вариантов сохраняются.
4. Артикул ссылки разрешается через `sku`, `group_id`, `sales_sku` и короткий ключ карточки.
5. Неразрешённые ссылки не попадают в публичную выдачу, но видны в preview и метриках warm-status.
6. Полная generation публикуется atomic swap; предыдущая продолжает обслуживать API до успешной замены.

## Настройки

- `enabled` — включает публичную выдачу;
- `attribute_keys` — допустимые названия атрибута комплектности;
- `max_items` — максимум ID в ответе, 1–40;
- `include_reverse_links` — если A явно ссылается на B, разрешить показывать A для B;
- `updated_reason` — комментарий к изменению.

Конфиг хранится в `runtime_settings.outfit_strategy` и не связан с `similar_strategy`.

## API

- `GET /api/v1/client/products/:id/outfits`;
- `GET|POST /api/v1/client/products/outfits`;
- `GET|PUT /api/admin/outfit-strategy`;
- `POST /api/admin/outfit-strategy/reset`;
- `POST /api/admin/outfit-strategy/preview`;
- `GET /api/admin/outfit-strategy/warm-status`.

Публичный ответ содержит только ID. Preview возвращает карточки и диагностику `resolved`, `not_found`, `self_reference`.

## Следующий этап

Итоговая витрина доступна через независимый слой композиции:

- `GET /api/v1/client/products/:id/recommendations`;
- `GET|POST /api/v1/client/products/recommendations`;
- `show_outfit_first=false` по умолчанию;
- `serve_composed_via_similar=false` по умолчанию и позволяет управляемо отдать эту же композицию через старую ручку `/similar`;
- при включении все валидные комплектные ID ставятся перед текущей выдачей `similar`;
- дубли, базовый товар и `exclude_product_ids` удаляются, общий `limit` сохраняется;
- при недоступности `outfits` композиция деградирует до обычных похожих;
- объединение выполняется поверх двух serving snapshot и не требует новой generation или прогрева.

Конфиг хранится отдельно в `runtime_settings.recommendation_composition`. При выключенном `serve_composed_via_similar` старый контракт `/similar` остаётся независимым; включение флага меняет только маршрутизацию ответа и служит быстрым rollout/rollback-механизмом.
