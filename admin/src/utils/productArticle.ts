/** Артикул цветоразмера из фида (offer.id), напр. GKT024636-1. */
export function productColorSizeArticle(item: {
  id?: string;
  sku?: string;
} | null | undefined): string {
  if (!item) return '';
  const id = (item.id ?? '').trim();
  const sku = (item.sku ?? '').trim();
  const idIsVariant = /-\d+$/.test(id);
  const skuIsVariant = /-\d+$/.test(sku);
  if (idIsVariant && !skuIsVariant) return id;
  if (skuIsVariant && !idIsVariant) return sku;
  if (id && id !== sku) return id;
  return id || sku;
}

const PRODUCT_SITE_ORIGIN = (
  process.env.REACT_APP_PRODUCT_SITE_ORIGIN || 'https://shop.example.com'
).replace(/\/$/, '');

/** Ссылка на карточку товара на настраиваемой витрине. */
export function productShopHref(item: { id?: string; sku?: string }): string {
  const article = productColorSizeArticle(item);
  if (!article) return PRODUCT_SITE_ORIGIN;
  return `${PRODUCT_SITE_ORIGIN}/product/${encodeURIComponent(article)}`;
}
