# Backend

Go service for importing a product XML feed and serving similar-item and product-set recommendations.

## Local development

```bash
cp .env.example .env
# Set FEED_URL.
make pg-up
make run
```

The API listens on `:8080` by default. PostgreSQL is isolated from the original project and uses port `5435` in this repository's Compose configuration.

Configuration is provided through environment variables: `DATABASE_URL`, `FEED_URL`, `FEED_SYNC_INTERVAL`, `WARM_TIMEOUT`, `WARM_WORKERS` and `PORT`.
