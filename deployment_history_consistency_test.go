package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestDeploymentHistoryConcurrentReports(t *testing.T) {
	for _, query := range []string{"limit=200", "project=app&limit=200", "environment=prod&limit=200"} {
		t.Run(query, func(t *testing.T) {
			t.Setenv("API_TOKEN", "")
			handler := deploymentTestServer(t, "")
			// The single-connection test DB forces the reader and writer to interleave
			// between queries unless the entire response is read in a transaction.
			writerDone := make(chan error, 1)
			go func() {
				for i := 0; i < 150; i++ {
					_, err := db.Exec(`INSERT INTO deployments (project, git_url, environment, deployed_at)
						VALUES ('app', 'https://git.example/commit', 'prod', ?)`,
						fmt.Sprintf("2026-10-20T12:00:00.%09dZ", i))
					if err != nil {
						writerDone <- err
						return
					}
				}
				writerDone <- nil
			}()
			// Always join before cleanup closes the DB, even if an assertion fails.
			defer func() {
				if err := <-writerDone; err != nil {
					t.Errorf("incoming report failed: %v", err)
				}
			}()
			for i := 0; i < 100; i++ {
				w := deploymentRequest(handler, "GET", "/api/deployments?"+query, "", "")
				if w.Code != http.StatusOK {
					t.Fatalf("history request failed: %d %s", w.Code, w.Body.String())
				}
				var result struct {
					Deployments  []Deployment `json:"deployments"`
					Total        int          `json:"total"`
					Environments []string     `json:"environments"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Total != len(result.Deployments) {
					t.Fatalf("inconsistent snapshot: total=%d, rows=%d", result.Total, len(result.Deployments))
				}
				if result.Total == 0 && len(result.Environments) != 0 {
					t.Fatal("environment list includes a report outside the page snapshot")
				}
				if result.Total > 0 && (len(result.Environments) != 1 || result.Environments[0] != "prod") {
					t.Fatalf("environment list missing from snapshot: %v", result.Environments)
				}
			}
		})
	}
}

func TestDeploymentHistoryIndexes(t *testing.T) {
	deploymentTestServer(t, "")
	// Index creation must also be safe on subsequent startup migrations.
	if err := initDeployments(db); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		where string
		index string
	}{
		{"", "idx_deployments_time"},
		{" WHERE environment = 'prod'", "idx_deployments_env_time"},
		{" WHERE project = 'app'", "idx_deployments_project_time"},
		{" WHERE project = 'app' AND environment = 'prod'", "idx_deployments_project_env_time"},
	} {
		t.Run(tc.index, func(t *testing.T) {
			rows, err := db.Query("EXPLAIN QUERY PLAN SELECT id, project, git_url, environment, deployed_at, version, deployment_url, version_created_at FROM deployments" +
				tc.where + " ORDER BY deployed_at DESC, id DESC LIMIT 50 OFFSET 50")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = rows.Close() }()
			plan := ""
			for rows.Next() {
				var id, parent, unused int
				var detail string
				if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
					t.Fatal(err)
				}
				plan += detail + "\n"
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(plan, tc.index) || strings.Contains(plan, "TEMP B-TREE") {
				t.Fatalf("history should use %s without a temporary sort; plan:\n%s", tc.index, plan)
			}
		})
	}
}
