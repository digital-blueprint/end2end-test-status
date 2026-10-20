package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
)

func TestDeploymentURL(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	handler := deploymentTestServer(t, "")
	w := deploymentRequest(handler, "POST", "/webhook/deployments", `{
		"project":"app", "environment":"prod", "git_url":"https://git.example/commit/abc123",
		"version":"v1.2.3", "deployment_url":" https://app.example.com/ "
	}`, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("deployment URL report failed: %d %s", w.Code, w.Body.String())
	}
	for _, path := range []string{"/api/deployments", "/api/deployments?project=app"} {
		w = deploymentRequest(handler, "GET", path, "", "")
		var result struct {
			Deployments []Deployment `json:"deployments"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Deployments) != 1 || result.Deployments[0].DeploymentURL != "https://app.example.com/" {
			t.Fatalf("URL not preserved in %s: %+v", path, result)
		}
	}
	w = deploymentRequest(handler, "GET", "/api/deployments/projects", "", "")
	var overview struct {
		Projects []DeploymentProject `json:"projects"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if len(overview.Projects) != 1 || len(overview.Projects[0].Environments) != 1 || overview.Projects[0].Environments[0].DeploymentURL != "https://app.example.com/" {
		t.Fatalf("URL not preserved in overview: %+v", overview)
	}
	for _, invalid := range []string{"javascript:alert(1)", "ftp://app.example.com", "/relative", "https://user:secret@app.example.com/"} {
		body, err := json.Marshal(DeploymentPayload{Project: "app", Environment: "prod", GitURL: "https://git.example/app", DeploymentURL: invalid})
		if err != nil {
			t.Fatal(err)
		}
		w := deploymentRequest(handler, "POST", "/webhook/deployments", string(body), "")
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for deployment URL %q, got %d", invalid, w.Code)
		}
	}
}

func TestDeploymentURLMigrationWithExistingVersion(t *testing.T) {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	database.SetMaxOpenConns(1)
	_, err = database.Exec(`CREATE TABLE deployments (
		id INTEGER PRIMARY KEY AUTOINCREMENT, project TEXT NOT NULL, git_url TEXT NOT NULL,
		environment TEXT NOT NULL, deployed_at TEXT NOT NULL, version TEXT NOT NULL DEFAULT ''
	);
	INSERT INTO deployments (project, git_url, environment, deployed_at, version)
	VALUES ('app', 'https://git.example/old', 'prod', '2026-10-20T12:00:00.000000000Z', 'v1.0.0');`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := initDeployments(database); err != nil {
			t.Fatal(err)
		}
	}
	var version, deploymentURL, versionCreatedAt string
	if err := database.QueryRow("SELECT version, deployment_url, version_created_at FROM deployments").Scan(&version, &deploymentURL, &versionCreatedAt); err != nil {
		t.Fatal(err)
	}
	if version != "v1.0.0" || deploymentURL != "" || versionCreatedAt != "" {
		t.Fatalf("migration changed existing metadata: version=%q, URL=%q", version, deploymentURL)
	}
}
