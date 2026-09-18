import { productColorSizeArticle, productShopHref } from './productArticle';

describe('productArticle', () => {
  it('предпочитает id (цветомодель) перед group sku', () => {
    expect(productColorSizeArticle({ id: 'GKT024636-1', sku: 'GRT000848' })).toBe('GKT024636-1');
  });

  it('берёт sku с суффиксом -N если id совпадает с group', () => {
    expect(productColorSizeArticle({ id: 'GRT000848', sku: 'GKT024636-1' })).toBe('GKT024636-1');
  });

  it('fallback на единственное доступное значение', () => {
    expect(productColorSizeArticle({ sku: 'GRT000823' })).toBe('GRT000823');
  });

  it('формирует ссылку на карточку на настроенной витрине', () => {
    expect(productShopHref({ id: 'GKT024636-1', sku: 'GRT000848' })).toBe(
      'https://shop.example.com/product/GKT024636-1'
    );
  });
});
