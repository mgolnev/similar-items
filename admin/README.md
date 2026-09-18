# Admin UI

React admin UI for configuring recommendation strategies, loading a product feed and checking API results.

## Local development

```bash
npm ci
npm start
```

The development server listens on `:3000` and proxies `/api` to `http://localhost:8080`. Set `REACT_APP_API_URL` to use another backend and `REACT_APP_PRODUCT_SITE_ORIGIN` to control storefront product links.
