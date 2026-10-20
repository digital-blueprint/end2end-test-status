package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectsLatestStatusUsesNewestResult(t *testing.T) {
	deploymentTestServer(t, "")

	// All rows share the same created_at second, so the newest id must win.
	_, err := db.Exec(`INSERT INTO test_results (project, spec, browser, status, pipeline_id, job_id, job_url, created_at) VALUES
		('shop', 's', 'chrome', 'success', '', '', '', '2026-10-20 10:00:00'),
		('shop', 's', 'chrome', 'failed',  '', '', '', '2026-10-20 10:00:00'),
		('billing', 's', 'chrome', 'failed',  '', '', '', '2026-10-20 10:00:00'),
		('billing', 's', 'chrome', 'success', '', '', '', '2026-10-20 10:00:00'),
		('docs', 's', 'chrome', 'failed',  '', '', '', '2026-10-20 11:00:00'),
		('docs', 's', 'chrome', 'success', '', '', '', '2026-10-20 09:00:00')`)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	projectsHandler(w, httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var summaries []ProjectSummary
	if err := json.Unmarshal(w.Body.Bytes(), &summaries); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"shop": "failed", "billing": "success", "docs": "failed"}
	if len(summaries) != len(want) {
		t.Fatalf("got %d projects, want %d", len(summaries), len(want))
	}
	for _, s := range summaries {
		if s.LatestStatus != want[s.Project] {
			t.Errorf("%s: latest_status = %q, want %q", s.Project, s.LatestStatus, want[s.Project])
		}
		if s.TotalRuns != 2 || s.PassedRuns != 1 || s.FailedRuns != 1 {
			t.Errorf("%s: runs = %d/%d/%d, want 2/1/1", s.Project, s.TotalRuns, s.PassedRuns, s.FailedRuns)
		}
	}
}
