package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAllDeploymentHistory(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	handler := deploymentTestServer(t, "")
	recordDeployment(t, handler, "frontend", "dev", "https://git.example/one", "2026-10-20T12:00:00Z", "v1.0.0")
	recordDeployment(t, handler, "backend", "prod", "https://git.example/two", "2026-10-20T13:00:00Z", "v2.0.0")
	recordDeployment(t, handler, "frontend", "prod", "https://git.example/three", "2026-10-20T14:00:00Z")
	for _, tc := range []struct {
		query      string
		total      int
		firstURL   string
		firstCount int
	}{
		{"limit=1&offset=1", 3, "https://git.example/two", 1},
		{"environment=prod", 2, "https://git.example/three", 2},
		{"", 3, "https://git.example/three", 3},
	} {
		w := deploymentRequest(handler, "GET", "/api/deployments?"+tc.query, "", "")
		var result struct {
			Deployments  []Deployment `json:"deployments"`
			Total        int          `json:"total"`
			Environments []string     `json:"environments"`
		}
		if w.Code != http.StatusOK {
			t.Fatalf("all deployments failed: %d %s", w.Code, w.Body.String())
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Total != tc.total || len(result.Deployments) != tc.firstCount || result.Deployments[0].GitURL != tc.firstURL || len(result.Environments) != 2 {
			t.Fatalf("wrong global history for %q: %+v", tc.query, result)
		}
	}
}
