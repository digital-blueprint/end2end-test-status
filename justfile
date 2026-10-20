# Use `just <recipe>` to run a recipe
# https://just.systems/man/en/

import ".shared/common.just"

# By default, run the `--list` command
default:
    @just --list

# Build the docker image without cache
docker-build:
    docker compose build --no-cache

# Send demo E2E test results to a running server (defaults match docker-compose.yml)
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
