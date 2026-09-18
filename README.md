# Similar Items for Clever

Independent recommendation service for similar products and product sets. This repository is a standalone monorepo and has no runtime dependency on the original GitLab projects.

## Structure

- `backend/` — Go API, feed import, scoring, caches and PostgreSQL migrations.
- `admin/` — React admin UI and recommendation preview.
- `docker-compose.yml` — local PostgreSQL, backend and admin UI.

## Quick start

```bash
cp .env.example .env
# Set FEED_URL and PRODUCT_SITE_ORIGIN in .env.
docker compose up --build
```

After startup:

- Admin UI: <http://localhost:3000>
- API: <http://localhost:8080>
- Health: <http://localhost:8080/health>

The product feed is intentionally not bundled. `FEED_URL` must point to the Clever feed. Local and production credentials belong only in `.env` or the deployment secret store; `.env` is ignored by Git.

## Independent connections

The project uses its own PostgreSQL volume (`clever_similar_postgres`), Docker service names and ports. Storefront links are controlled by `PRODUCT_SITE_ORIGIN`. The source repository has only one remote: `git@github.com:mgolnev/similar-items.git`.

## Maintenance note

The admin application currently retains the original Create React App dependency stack. Builds and tests pass, but `npm audit` reports inherited transitive vulnerabilities. Dependabot is enabled; migrating the admin UI to a current build tool should be handled as a dedicated follow-up rather than with a breaking `npm audit fix --force`.
