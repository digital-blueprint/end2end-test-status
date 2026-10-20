# end2end-test-status

A lightweight web service that receives end-to-end test results and successful deployment reports via webhooks, stores them in SQLite, and displays them through a Vue.js dashboard.

## How it works

1. CI jobs POST test results to `/webhook` or successful deployments to `/webhook/deployments`.
2. The Go server authenticates requests and stores reports in SQLite.
3. The Vue.js UI, served by the same binary, shows separate E2E Tests and Deployments dashboards with project overviews and history.

## Docker

The published image is `ghcr.io/digital-blueprint/end2end-test-status`.

```sh
docker run -d \
  -p 8080:8080 \
  -v /path/to/data:/data \
  -e API_TOKEN=your-secret-token \
  -e PATH_PREFIX=/end2end-test-status \
  -e FOOTER_TEXT="QA dashboard" \
  ghcr.io/digital-blueprint/end2end-test-status
```

The UI is available at `http://localhost:8080`. When hosting under a sub-path, include the prefix in the URL (e.g. `https://your-server/end2end-test-status/`). For static hosting, configure the server to serve `index.html` for SPA routes under that prefix.

### docker-compose example

```yaml
services:
  e2e-status:
    image: ghcr.io/digital-blueprint/end2end-test-status:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      API_TOKEN: your-secret-token
      PATH_PREFIX: /end2end-test-status
      FOOTER_TEXT: QA dashboard
    restart: unless-stopped
```

## Environment variables

| Variable      | Default           | Description                                                                                                                    |
| ------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `API_TOKEN`   | _(none)_          | Bearer token required on both webhook endpoints. If unset, the webhooks accept all requests.                                   |
| `PORT`        | `8080`            | Port the HTTP server listens on.                                                                                               |
| `PATH_PREFIX` | _(none)_          | Optional URL path prefix for hosting under a sub-path (e.g. `/end2end-test-status`). Used by the Go server and the Vite build. |
| `FOOTER_TEXT` | _(none)_          | Optional footer text shown in the UI.                                                                                          |
| `DB_PATH`     | `/data/db.sqlite` | Path to the SQLite database file. The parent directory is created if missing.                                                  |

By default the SQLite database is stored at `/data/db.sqlite`. Mount `/data` as a persistent volume.

## Webhook

GitLab CI jobs send results with the webhook endpoint. If `PATH_PREFIX` is set, include the prefix in the URL (e.g. `/end2end-test-status/webhook`).

```sh
curl -s -X POST "https://your-host/webhook" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_TOKEN" \
  -d "{
    \"project\": \"$NAME\",
    \"spec\": \"$SPEC\",
    \"browser\": \"$BROWSER\",
    \"status\": \"$CI_JOB_STATUS\",
    \"pipeline_id\": \"$CI_PIPELINE_ID\",
    \"job_id\": \"$CI_JOB_ID\",
    \"job_url\": \"$CI_JOB_URL\"
  }"
```

### Payload fields

| Field         | Required | Description                                  |
| ------------- | -------- | -------------------------------------------- |
| `project`     | yes      | Project name (e.g. `my-app`)                 |
| `status`      | yes      | Job status — typically `success` or `failed` |
| `spec`        | no       | Playwright spec file path                    |
| `browser`     | no       | Browser used (e.g. `chromium`, `firefox`)    |
| `pipeline_id` | no       | GitLab `$CI_PIPELINE_ID`                     |
| `job_id`      | no       | GitLab `$CI_JOB_ID`                          |
| `job_url`     | no       | GitLab `$CI_JOB_URL` — linked in the UI      |

Returns `201 Created` on success.

## Deployment webhook

After a deployment succeeds, POST to `/webhook/deployments`. Include `PATH_PREFIX` in the URL if configured, and use the same API token as the test webhook.

```sh
curl --fail-with-body -X POST "https://your-host/webhook/deployments" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_TOKEN" \
  -d '{
    "project": "my-app",
    "git_url": "https://gitlab.example.com/team/my-app/-/commit/abc123",
    "environment": "staging",
    "version": "v1.2.3-4-gabc123",
    "version_created_at": "2026-10-13T12:30:00Z",
    "deployment_url": "https://staging.my-app.example.com/",
    "deployed_at": "2026-10-20T12:30:00Z"
  }'
```

| Field                | Required | Description                                                                                           |
| -------------------- | -------- | ----------------------------------------------------------------------------------------------------- |
| `project`            | yes      | Deployment project name, independent of the E2E test project list                                     |
| `git_url`            | yes      | HTTP(S) Git URL without embedded credentials; a commit-specific link is recommended                   |
| `environment`        | yes      | Freeform, case-sensitive name, e.g. `dev`, `demo`, `staging`, `prod`, or `preview/42`                 |
| `deployed_at`        | no       | When the deployment happened, not when the report was received; defaults to the server's current time |
| `version`            | no       | Freeform version label, e.g. `v1.2.3`, `v1.2.3-4-gabc123`, or a commit hash                           |
| `version_created_at` | no       | When the deployed version, tag, or revision was created; independent of `deployed_at`                 |
| `deployment_url`     | no       | HTTP(S) URL of the running deployment, without embedded credentials                                   |

Both timestamps must use RFC3339 with a timezone, such as `2026-10-20T12:30:00Z`, and are stored in UTC. Project, environment, URL, and version values are trimmed. Each accepted report returns `201 Created` and adds a history entry; retries are not deduplicated. The latest deployment is selected by `deployed_at`, with insertion order breaking ties, so delayed reports do not replace newer deployments.

### Reporting from GitLab CI

Run this in the deployed checkout after a successful deployment. It requires `jq`, uses `git describe --tags --always` for the version label, and uses the commit's committer date for `version_created_at`. Omit `deployed_at` to use the report time.

```sh
jq -n \
  --arg project "$NAME" \
  --arg git_url "$CI_PROJECT_URL/-/commit/$CI_COMMIT_SHA" \
  --arg environment "$DEPLOY_ENV" \
  --arg version "$(git describe --tags --always)" \
  --arg version_created_at "$(git log -1 --format=%cI)" \
  --arg deployment_url "$DEPLOY_URL" \
  '{project: $project, git_url: $git_url, environment: $environment, version: $version, version_created_at: $version_created_at, deployment_url: $deployment_url}' | \
  curl --fail-with-body -X POST "https://your-host/webhook/deployments" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $API_TOKEN" \
    --data-binary @-
```

Fetch tags and sufficient history in CI for tag-based version labels; `--always` falls back to a commit hash. To report a tag's age instead of the commit's, use `git for-each-ref --format='%(creatordate:iso-strict)' refs/tags/v1.2.3`. This uses the tagger date for annotated tags and the commit date for lightweight tags. The server does not fetch Git metadata.

### Dashboard

Deployment projects are independent of E2E test projects and appear after their first report. **Overview** shows each project's deployed environments, newest first; **All Deployments** shows paginated history across all projects and environments, without an environment selector. Project cards open project history, where environment filtering is available; environment names open history filtered to that environment. Project environment filters persist in the URL. **Open environment** opens the current environment URL, not a historical deployment.

Versions link to their Git URLs and optionally show their creation age, e.g. `v1.2.3 (created 3 days ago)`. Dates update every 30 seconds; hover for exact local timestamps. Version tooltips also show the age at deployment, e.g. “Deployed 7 days after creation”.

Projects with multiple environments have two selectors on the deployment overview, defaulting to the last displayed environment on the left and the first on the right. **⇄** swaps them. **Compare** opens a GitHub or GitLab comparison of their latest deployed commits in a new tab. Both deployments must have commit-specific Git URLs from the same repository; repository-only or unsupported URLs disable comparison. Comparison is also disabled when both environments point to the same commit, with a message explaining that there are no changes to compare. Self-hosted GitLab and GitHub-style commit URLs are supported. No Git API calls or backend configuration are needed.

Deployment tables and optional fields are created or migrated automatically in the existing SQLite database without changing test results. Older records have empty optional fields.

## API

All API endpoints return JSON and are public. Include `PATH_PREFIX` in their URLs if configured.

| Method | Path                              | Description                                                  |
| ------ | --------------------------------- | ------------------------------------------------------------ |
| `GET`  | `/api/projects`                   | All projects with pass/fail counts and latest status         |
| `GET`  | `/api/projects/{project}/results` | Results for a single project                                 |
| `GET`  | `/api/results`                    | All results across all projects                              |
| `GET`  | `/api/health`                     | Health check                                                 |
| `GET`  | `/api/deployments/projects`       | Projects with latest deployments per environment             |
| `GET`  | `/api/deployments`                | Paginated deployment history, optionally filtered by project |

### Deployment endpoints

`GET /api/deployments/projects` returns `{ "projects": [{ "project": "my-app", "environments": [...] }] }`. Each project contains its latest record per environment; only projects with deployment reports are included.

`GET /api/deployments` returns `{ "deployments": [...], "total": 123, "environments": ["staging"] }`, newest first. Records contain `id` and the deployment payload fields; omitted optional fields are empty strings. `total` counts matching records before pagination. `environments` lists all environments for the selected project, or all reported environments when no project is selected, regardless of pagination or environment filtering.

| Parameter     | Description                                                                                |
| ------------- | ------------------------------------------------------------------------------------------ |
| `project`     | Optional, exact project name (omit for all projects; an explicitly empty value is invalid) |
| `environment` | Optional, exact environment name                                                           |
| `limit`       | Page size, 1–200 (default 50)                                                              |
| `offset`      | Non-negative number of records to skip (default 0)                                         |

URL-encode project and environment query values.

### Query parameters for result endpoints

| Parameter | Description                                           |
| --------- | ----------------------------------------------------- |
| `status`  | Filter by status (`success` / `failed`)               |
| `browser` | Filter by browser                                     |
| `spec`    | Substring filter on spec path (project endpoint only) |
| `limit`   | Max rows to return (default: `100` / `200`)           |

## Local development

Run `just term` to open a [Zellij](https://zellij.dev) session (layout in `term.kdl`) with lazygit, the backend, the Vite dev server, and a pane that seeds demo data with `just seed`. Open http://localhost:5173 once everything is running.

### Backend

```sh
# Requires Go 1.25+ and gcc (for CGO/sqlite3)
DB_PATH=./tmp/db.sqlite API_TOKEN=dev-token PORT=8080 go run .
# or: just backend
```

The server expects `frontend/dist/` to exist. Build the frontend first, or run both together.

### Frontend

```sh
cd frontend
npm install
npm run build   # set PATH_PREFIX for sub-path hosting
npm run dev   # Vite dev server on :5173, proxies /api and /webhook to :8080
npm test      # Date-formatting tests (requires Node.js 22.6+)
```

### Build frontend for embedding

```sh
cd frontend
PATH_PREFIX=/end2end-test-status npm run build
```

### Build Docker image with prefixed assets

```sh
docker build --build-arg PATH_PREFIX=/end2end-test-status -t ghcr.io/digital-blueprint/end2end-test-status:latest .

# Footer text is set at runtime via FOOTER_TEXT, so the published image can be reused.
```

This writes to `frontend/dist/`, which is embedded into the Go binary at compile time via `//go:embed`.

## CI / Docker image

The GitHub Actions workflow at `.github/workflows/docker.yml` builds and pushes the image to GHCR on every push to `main` and on version tags (`v*`).

Tags produced:

| Git ref       | Image tag                      |
| ------------- | ------------------------------ |
| `main` branch | `latest`, `main`, `sha-<hash>` |
| `v1.2.3` tag  | `1.2.3`, `1.2`, `sha-<hash>`   |
| Pull request  | build only, no push            |

The image is built for `linux/amd64` and `linux/arm64`.
