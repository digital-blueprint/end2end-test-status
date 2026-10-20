# Use `just <recipe>` to run a recipe
# https://just.systems/man/en/

import ".shared/common.just"

# By default, run the `--list` command
default:
    @just --list

# Variables

zellijSession := "end2end-test-status"

# Keep the default session name for the main checkout and distinguish linked worktrees.
zellijWorktreeSuffix := `if git_dir=$(git rev-parse --path-format=absolute --git-dir 2>/dev/null) && common_dir=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null) && [ "$git_dir" != "$common_dir" ]; then worktree_name=$(basename "$(git rev-parse --show-toplevel)"); printf -- '-%s' "$worktree_name" | tr -c '[:alnum:]_-' '-'; fi`
zellijSessionName := zellijSession + zellijWorktreeSuffix

# Open a terminal with the end2end-test-status session
[group('dev')]
term-run:
    zellij --layout term.kdl attach {{ zellijSessionName }} -c

# Kill the end2end-test-status session
[group('dev')]
term-kill:
    -zellij delete-session {{ zellijSessionName }} -f

# Kill and run a terminal with the end2end-test-status session
[group('dev')]
term: term-kill term-run

# Run the Go backend on :8080 with a local database in ./tmp
[group('dev')]
backend:
    #!/usr/bin/env bash
    set -euo pipefail
    # The backend embeds frontend/dist, so it must exist before compiling
    if [ ! -f frontend/dist/index.html ]; then
        (cd frontend && npm install && npm run build)
    fi
    DB_PATH=./tmp/db.sqlite API_TOKEN=dev-token PORT=8080 FOOTER_TEXT="Local development" go run .

# Run the Vite dev server on :5173 (proxies /api and /webhook to :8080)
[group('dev')]
frontend:
    cd frontend && npm install && npm run dev

# Build the docker image without cache
docker-build:
    docker compose build --no-cache

# Send demo E2E test results to a running server (defaults match docker-compose.yml and `just backend`)
[group('dev')]
seed url='http://localhost:8080' token='dev-token':
    #!/usr/bin/env bash
    set -euo pipefail
    post() {
        curl --fail-with-body -sS -o /dev/null -X POST "{{ url }}/webhook" \
            -H "Authorization: Bearer {{ token }}" \
            -H "Content-Type: application/json" \
            -d "$(printf '{"project":"%s","spec":"cypress/e2e/%s.cy.ts","browser":"%s","status":"%s","pipeline_id":"%s","job_id":"%s"}' \
                "$1" "$2" "$3" "$4" "$RANDOM" "$RANDOM")"
    }
    # The last result per project determines its latest status
    post shop-frontend checkout chrome success
    post shop-frontend checkout chrome failed
    post admin-portal login firefox success
    post admin-portal users chrome success
    post billing-api invoices chrome failed
    post billing-api invoices chrome success
    post docs-site search chrome canceled
    post user-service signup firefox success
    post search-service query chrome failed
    echo "Seeded 6 projects (3 passing, 3 failing) at {{ url }}"
