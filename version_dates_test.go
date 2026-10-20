package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestDeploymentVersionCreatedAt(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	handler := deploymentTestServer(t, "")
	w := deploymentRequest(handler, "POST", "/webhook/deployments", `{
		"project":"app", "environment":"prod", "git_url":"https://git.example/commit/abc123",
		"version":"v1.2.3", "deployed_at":"2026-10-20T12:00:00Z", "version_created_at":"2026-10-13T14:00:00+02:00"
	}`, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("version date report failed: %d %s", w.Code, w.Body.String())
	}
	for _, path := range []string{"/api/deployments", "/api/deployments?project=app"} {
		w := deploymentRequest(handler, "GET", path, "", "")
		var result struct {
			Deployments []Deployment `json:"deployments"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Deployments) != 1 || result.Deployments[0].VersionCreatedAt != "2026-10-13T12:00:00.000000000Z" || result.Deployments[0].DeployedAt != "2026-10-20T12:00:00.000000000Z" {
			t.Fatalf("version and deployment dates not independently preserved in %s: %+v", path, result)
		}
	}
	w = deploymentRequest(handler, "GET", "/api/deployments/projects", "", "")
	var overview struct {
		Projects []DeploymentProject `json:"projects"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if len(overview.Projects) != 1 || len(overview.Projects[0].Environments) != 1 || overview.Projects[0].Environments[0].VersionCreatedAt != "2026-10-13T12:00:00.000000000Z" {
		t.Fatalf("version date not returned in overview: %+v", overview)
	}
	for _, invalid := range []string{"not a date", "2026-10-13T12:00:00"} {
		body, err := json.Marshal(DeploymentPayload{Project: "app", Environment: "prod", GitURL: "https://git.example/app", VersionCreatedAt: invalid})
		if err != nil {
			t.Fatal(err)
		}
		w := deploymentRequest(handler, "POST", "/webhook/deployments", string(body), "")
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for version_created_at %q, got %d", invalid, w.Code)
		}
	}
	recordDeployment(t, handler, "app", "dev", "https://git.example/old", "2026-10-20T13:00:00Z")
	var createdAt string
	if err := db.QueryRow("SELECT version_created_at FROM deployments WHERE environment = 'dev'").Scan(&createdAt); err != nil {
		t.Fatal(err)
	}
	if createdAt != "" {
		t.Fatal("missing version date must stay empty, not default to deployment time")
	}
}
