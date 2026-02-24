# end2end-test-status

A lightweight web service that receives end-to-end test results from GitLab CI via webhook, stores them in SQLite, and displays them through a Vue.js dashboard.

## How it works

1. GitLab CI jobs run Playwright tests and POST results to `/webhook`
2. The Go server authenticates the request, stores the result in SQLite
3. The Vue.js UI (served by the same binary) shows a per-project dashboard and result history

## Docker

The published image is `ghcr.io/digital-blueprint/end2end-test-status`.

```sh
docker run -d \
  -p 8080:8080 \
  -v /path/to/data:/data \
  -e API_TOKEN=your-secret-token \
  -e PATH_PREFIX=/end2end-test-status \
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
    restart: unless-stopped
```

## Environment variables

| Variable      | Default  | Description                                                                                                                    |
| ------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `API_TOKEN`   | _(none)_ | Bearer token required on `POST /webhook`. If unset, the webhook accepts all requests.                                          |
| `PORT`        | `8080`   | Port the HTTP server listens on.                                                                                               |
| `PATH_PREFIX` | _(none)_ | Optional URL path prefix for hosting under a sub-path (e.g. `/end2end-test-status`). Used by the Go server and the Vite build. |

The SQLite database is stored at `/data/db.sqlite`. Mount `/data` as a persistent volume.

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

## API

All API endpoints return JSON.

| Method | Path                              | Description                                          |
| ------ | --------------------------------- | ---------------------------------------------------- |
| `GET`  | `/api/projects`                   | All projects with pass/fail counts and latest status |
| `GET`  | `/api/projects/{project}/results` | Results for a single project                         |
| `GET`  | `/api/results`                    | All results across all projects                      |
| `GET`  | `/api/health`                     | Health check                                         |

### Query parameters for result endpoints

| Parameter | Description                                           |
| --------- | ----------------------------------------------------- |
| `status`  | Filter by status (`success` / `failed`)               |
| `browser` | Filter by browser                                     |
| `spec`    | Substring filter on spec path (project endpoint only) |
| `limit`   | Max rows to return (default: `100` / `200`)           |

## Local development

### Backend

```sh
# Requires Go 1.25+ and gcc (for CGO/sqlite3)
API_TOKEN=dev PORT=8080 go run .
```

The server expects `frontend/dist/` to exist. Build the frontend first, or run both together.

### Frontend

```sh
cd frontend
npm install
npm run build   # set PATH_PREFIX for sub-path hosting
npm run dev   # Vite dev server on :5173, proxies /api and /webhook to :8080
```

### Build frontend for embedding

```sh
cd frontend
PATH_PREFIX=/end2end-test-status npm run build
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
