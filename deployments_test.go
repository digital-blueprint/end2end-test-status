package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func deploymentTestServer(t *testing.T, prefix string) http.Handler {
	t.Helper()
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	previous := db
	db = database
	t.Cleanup(func() { _ = database.Close(); db = previous })
	_, err = database.Exec(`CREATE TABLE test_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT, project TEXT, spec TEXT, browser TEXT, status TEXT,
		pipeline_id TEXT, job_id TEXT, job_url TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := initDeployments(database); err != nil {
		t.Fatal(err)
	}
	r := mux.NewRouter()
	r.HandleFunc(prefix+"/webhook", webhookHandler).Methods("POST")
	r.HandleFunc(prefix+"/webhook/deployments", deploymentWebhookHandler).Methods("POST")
	r.HandleFunc(prefix+"/api/deployments/projects", deploymentProjectsHandler).Methods("GET")
	r.HandleFunc(prefix+"/api/deployments", deploymentsHandler).Methods("GET")
	return authMiddleware(prefix, r)
}

func deploymentRequest(handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func recordDeployment(t *testing.T, handler http.Handler, project, environment, gitURL, deployedAt string, version ...string) {
	t.Helper()
	payload := DeploymentPayload{Project: project, Environment: environment, GitURL: gitURL, DeployedAt: deployedAt}
	if len(version) > 0 {
		payload.Version = version[0]
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	w := deploymentRequest(handler, "POST", "/webhook/deployments", string(body), "")
	if w.Code != http.StatusCreated {
		t.Fatalf("record deployment: %d %s", w.Code, w.Body.String())
	}
}

func TestDeploymentWebhook(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	handler := deploymentTestServer(t, "")
	before := time.Now().UTC()
	recordDeployment(t, handler, " app ", " feature/ABC-123 ", " https://git.example/app/commit/one ", "")
	var project, environment, gitURL, deployedAt string
	if err := db.QueryRow("SELECT project, environment, git_url, deployed_at FROM deployments").Scan(&project, &environment, &gitURL, &deployedAt); err != nil {
		t.Fatal(err)
	}
	if project != "app" || environment != "feature/ABC-123" || gitURL != "https://git.example/app/commit/one" {
		t.Fatalf("unexpected stored fields: %q %q %q", project, environment, gitURL)
	}
	parsed, err := time.Parse(time.RFC3339Nano, deployedAt)
	if err != nil || parsed.Before(before) || parsed.After(time.Now().UTC()) {
		t.Fatalf("invalid default deployment time: %q (%v)", deployedAt, err)
	}
	recordDeployment(t, handler, "app", "Demo EU", "https://git.example/app/commit/two", "2026-10-20T14:00:00.123+02:00")
	if err := db.QueryRow("SELECT deployed_at FROM deployments ORDER BY id DESC LIMIT 1").Scan(&deployedAt); err != nil {
		t.Fatal(err)
	}
	if deployedAt != "2026-10-20T12:00:00.123000000Z" {
		t.Fatalf("timestamp not normalized: %s", deployedAt)
	}
	// Startup migration is additive and safe to repeat.
	if err := initDeployments(db); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM deployments").Scan(&count); err != nil || count != 2 {
		t.Fatalf("migration lost history: %d, %v", count, err)
	}
}

func TestDeploymentValidation(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	handler := deploymentTestServer(t, "")
	for _, body := range []string{
		`{`, `null`, `{}`, `{"project":" ","environment":"prod","git_url":"https://git.example/app"}`,
		`{"project":"app","environment":" ","git_url":"https://git.example/app"}`,
		`{"project":"app","environment":"prod","git_url":" "}`,
		`{"project":"app","environment":"prod","git_url":"javascript:alert(1)"}`,
		`{"project":"app","environment":"prod","git_url":"git@git.example:app.git"}`,
		`{"project":"app","environment":"prod","git_url":"https://user:secret@git.example/app"}`,
		`{"project":"app","environment":"prod","git_url":"https://git.example/app","deployed_at":"2026-10-20T12:00:00"}`,
		`{"project":"app","environment":"prod","git_url":"https://git.example/app","deployed_at":"not a date"}`,
	} {
		w := deploymentRequest(handler, "POST", "/webhook/deployments", body, "")
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for %s, got %d", body, w.Code)
		}
	}
}

func TestDeploymentAuthenticationAndPrefix(t *testing.T) {
	t.Setenv("API_TOKEN", "secret")
	handler := deploymentTestServer(t, "/status")
	for _, endpoint := range []string{"/webhook", "/webhook/deployments"} {
		for _, token := range []string{"", "wrong"} {
			w := deploymentRequest(handler, "POST", "/status"+endpoint, `{}`, token)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected unauthorized for %s: %d", endpoint, w.Code)
			}
		}
	}
	w := deploymentRequest(handler, "POST", "/status/webhook/deployments",
		`{"project":"app","environment":"custom","git_url":"https://git.example/app"}`, "secret")
	if w.Code != http.StatusCreated {
		t.Fatalf("authenticated prefixed webhook failed: %d %s", w.Code, w.Body.String())
	}
	w = deploymentRequest(handler, "POST", "/status/webhook", `{"project":"tests","status":"success"}`, "secret")
	if w.Code != http.StatusCreated {
		t.Fatalf("existing webhook regression: %d", w.Code)
	}
	w = deploymentRequest(handler, "GET", "/status/api/deployments/projects", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("read API should remain public: %d", w.Code)
	}
}

func TestDeploymentOverview(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	handler := deploymentTestServer(t, "")
	if _, err := db.Exec("INSERT INTO test_results (project, status) VALUES ('tests-only', 'success')"); err != nil {
		t.Fatal(err)
	}
	recordDeployment(t, handler, "app", "prod", "https://git.example/new", "2026-10-20T12:00:00Z")
	// A late report must not replace the newer deployment.
	recordDeployment(t, handler, "app", "prod", "https://git.example/old", "2026-10-19T12:00:00Z")
	recordDeployment(t, handler, "app", "prod", "https://git.example/tie", "2026-10-20T12:00:00Z", "v1.2.3-4-gabc123")
	recordDeployment(t, handler, "app", "preview/42", "https://git.example/preview", "2026-10-20T12:00:00.1Z")
	recordDeployment(t, handler, "deployment-only", "QA EU", "https://git.example/qa", "2026-10-20T12:00:00Z")
	w := deploymentRequest(handler, "GET", "/api/deployments/projects", "", "")
	var result struct {
		Projects []DeploymentProject `json:"projects"`
	}
	if w.Code != 200 {
		t.Fatalf("overview failed: %s", w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Projects) != 2 || len(result.Projects[0].Environments) != 2 || len(result.Projects[1].Environments) != 1 {
		t.Fatalf("missing projects or per-project environments: %+v", result)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 || fields["projects"] == nil {
		t.Fatalf("overview must return only projects, not a global environment list: %s", w.Body.String())
	}
	for _, project := range result.Projects {
		if project.Project == "tests-only" {
			t.Fatal("E2E-only projects must not appear in the deployment overview")
		}
		for _, d := range project.Environments {
			if d.Environment == "prod" && (d.GitURL != "https://git.example/tie" || d.Version != "v1.2.3-4-gabc123") {
				t.Fatalf("wrong latest deployment: %+v", d)
			}
		}
	}
}

func TestDeploymentOverviewEmpty(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	handler := deploymentTestServer(t, "")
	w := deploymentRequest(handler, "GET", "/api/deployments/projects", "", "")
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"projects":[]}` {
		t.Fatalf("unexpected empty overview: %d %s", w.Code, w.Body.String())
	}
}

func TestDeploymentVersion(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	handler := deploymentTestServer(t, "")
	recordDeployment(t, handler, "app", "prod", "https://git.example/one", "2026-10-20T12:00:00Z", " v1.2.3-4-gabc123 ")
	recordDeployment(t, handler, "app", "dev", "https://git.example/two", "2026-10-20T13:00:00Z")
	w := deploymentRequest(handler, "GET", "/api/deployments?project=app", "", "")
	var result struct {
		Deployments []Deployment `json:"deployments"`
	}
	if w.Code != http.StatusOK {
		t.Fatalf("history failed: %s", w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Deployments) != 2 || result.Deployments[0].Version != "" || result.Deployments[1].Version != "v1.2.3-4-gabc123" {
		t.Fatalf("optional version not preserved/trimmed: %+v", result)
	}
}

func TestDeploymentVersionMigration(t *testing.T) {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	database.SetMaxOpenConns(1)
	_, err = database.Exec(`CREATE TABLE deployments (
		id INTEGER PRIMARY KEY AUTOINCREMENT, project TEXT NOT NULL, git_url TEXT NOT NULL,
		environment TEXT NOT NULL, deployed_at TEXT NOT NULL
	);
	INSERT INTO deployments (project, git_url, environment, deployed_at)
	VALUES ('app', 'https://git.example/old', 'prod', '2026-10-20T12:00:00.000000000Z');`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := initDeployments(database); err != nil {
			t.Fatal(err)
		}
	}
	var id int64
	var project, gitURL, environment, deployedAt, version, deploymentURL string
	if err := database.QueryRow("SELECT id, project, git_url, environment, deployed_at, version, deployment_url FROM deployments").
		Scan(&id, &project, &gitURL, &environment, &deployedAt, &version, &deploymentURL); err != nil {
		t.Fatal(err)
	}
	if id != 1 || project != "app" || gitURL != "https://git.example/old" || environment != "prod" || deployedAt != "2026-10-20T12:00:00.000000000Z" || version != "" || deploymentURL != "" {
		t.Fatal("migration must preserve the old record and default its version to empty")
	}
}

func TestDeploymentHistory(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	handler := deploymentTestServer(t, "")
	for i, env := range []string{"prod", "preview/42", "prod"} {
		recordDeployment(t, handler, "team/app 100%", env, fmt.Sprintf("https://git.example/%d", i), fmt.Sprintf("2026-10-20T12:00:0%dZ", i))
	}
	recordDeployment(t, handler, "other", "prod", "https://git.example/other", "2026-10-20T12:00:04Z")
	w := deploymentRequest(handler, "GET", "/api/deployments?project=team%2Fapp+100%25&environment=prod&limit=1&offset=1", "", "")
	var result struct {
		Deployments  []Deployment `json:"deployments"`
		Total        int          `json:"total"`
		Environments []string     `json:"environments"`
	}
	if w.Code != 200 {
		t.Fatalf("history failed: %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || len(result.Deployments) != 1 || result.Deployments[0].GitURL != "https://git.example/0" || len(result.Environments) != 2 {
		t.Fatalf("wrong filtered/paginated history: %+v", result)
	}
	for _, query := range []string{"project=", "project=app&limit=0", "project=app&limit=201", "project=app&limit=abc", "project=app&offset=-1", "project=app&offset=abc"} {
		w := deploymentRequest(handler, "GET", "/api/deployments?"+query, "", "")
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for %q: %d", query, w.Code)
		}
	}
	w = deploymentRequest(handler, "GET", "/api/deployments?project=unknown", "", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"deployments":[]`) || !strings.Contains(w.Body.String(), `"total":0`) {
		t.Fatalf("wrong empty history: %s", w.Body.String())
	}
}
