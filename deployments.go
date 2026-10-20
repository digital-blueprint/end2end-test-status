package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Fixed-width UTC timestamps preserve chronological ordering in SQLite.
const deploymentTimeFormat = "2006-01-02T15:04:05.000000000Z"

type DeploymentPayload struct {
	Project          string `json:"project"`
	GitURL           string `json:"git_url"`
	Environment      string `json:"environment"`
	DeployedAt       string `json:"deployed_at"`
	Version          string `json:"version"`
	DeploymentURL    string `json:"deployment_url"`
	VersionCreatedAt string `json:"version_created_at"`
}

type Deployment struct {
	ID               int64  `json:"id"`
	Project          string `json:"project"`
	GitURL           string `json:"git_url"`
	Environment      string `json:"environment"`
	DeployedAt       string `json:"deployed_at"`
	Version          string `json:"version"`
	DeploymentURL    string `json:"deployment_url"`
	VersionCreatedAt string `json:"version_created_at"`
}

type DeploymentProject struct {
	Project      string       `json:"project"`
	Environments []Deployment `json:"environments"`
}

func initDeployments(database *sql.DB) error {
	_, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS deployments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project TEXT NOT NULL,
			git_url TEXT NOT NULL,
			environment TEXT NOT NULL,
			deployed_at TEXT NOT NULL,
			version TEXT NOT NULL DEFAULT '',
			deployment_url TEXT NOT NULL DEFAULT '',
			version_created_at TEXT NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_deployments_project_time ON deployments(project, deployed_at DESC, id DESC);
		CREATE INDEX IF NOT EXISTS idx_deployments_project_env_time ON deployments(project, environment, deployed_at DESC, id DESC);
		CREATE INDEX IF NOT EXISTS idx_deployments_time ON deployments(deployed_at DESC, id DESC);
		CREATE INDEX IF NOT EXISTS idx_deployments_env_time ON deployments(environment, deployed_at DESC, id DESC);
	`)
	if err != nil {
		return err
	}
	// Add optional fields to existing tables without losing deployment history.
	rows, err := database.Query("PRAGMA table_info(deployments)")
	if err != nil {
		return err
	}
	columns := map[string]bool{}
	for rows.Next() {
		var id, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&id, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return err
		}
		columns[name] = true
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	for _, column := range []string{"version", "deployment_url", "version_created_at"} {
		if !columns[column] {
			if _, err := database.Exec("ALTER TABLE deployments ADD COLUMN " + column + " TEXT NOT NULL DEFAULT ''"); err != nil {
				return err
			}
		}
	}
	return nil
}

func deploymentWebhookHandler(w http.ResponseWriter, r *http.Request) {
	var payload DeploymentPayload
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	payload.Project = strings.TrimSpace(payload.Project)
	payload.Environment = strings.TrimSpace(payload.Environment)
	payload.GitURL = strings.TrimSpace(payload.GitURL)
	payload.Version = strings.TrimSpace(payload.Version)
	payload.DeploymentURL = strings.TrimSpace(payload.DeploymentURL)
	if payload.Project == "" || payload.Environment == "" || payload.GitURL == "" {
		http.Error(w, "project, environment and git_url are required", http.StatusBadRequest)
		return
	}
	if !validDeploymentURL(payload.GitURL) {
		http.Error(w, "git_url must be an HTTP(S) URL without credentials", http.StatusBadRequest)
		return
	}
	if payload.DeploymentURL != "" && !validDeploymentURL(payload.DeploymentURL) {
		http.Error(w, "deployment_url must be an HTTP(S) URL without credentials", http.StatusBadRequest)
		return
	}
	deployedAt := time.Now().UTC()
	var err error
	if payload.DeployedAt != "" {
		deployedAt, err = time.Parse(time.RFC3339Nano, payload.DeployedAt)
		if err != nil {
			http.Error(w, "deployed_at must be an RFC3339 timestamp with a timezone", http.StatusBadRequest)
			return
		}
	}
	if payload.VersionCreatedAt != "" {
		versionCreatedAt, err := time.Parse(time.RFC3339Nano, payload.VersionCreatedAt)
		if err != nil {
			http.Error(w, "version_created_at must be an RFC3339 timestamp with a timezone", http.StatusBadRequest)
			return
		}
		payload.VersionCreatedAt = versionCreatedAt.UTC().Format(deploymentTimeFormat)
	}
	_, err = db.Exec(`INSERT INTO deployments (project, git_url, environment, deployed_at, version, deployment_url, version_created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		payload.Project, payload.GitURL, payload.Environment, deployedAt.UTC().Format(deploymentTimeFormat), payload.Version, payload.DeploymentURL, payload.VersionCreatedAt)
	if err != nil {
		deploymentError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func deploymentProjectsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		WITH latest AS (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY project, environment ORDER BY deployed_at DESC, id DESC) AS rank
			FROM deployments
		)
		SELECT id, project, git_url, environment, deployed_at, version, deployment_url, version_created_at
		FROM latest WHERE rank = 1
		ORDER BY project, environment
	`)
	if err != nil {
		deploymentError(w, err)
		return
	}
	defer func() { _ = rows.Close() }()
	projects := []DeploymentProject{}
	for rows.Next() {
		var d Deployment
		if err := rows.Scan(&d.ID, &d.Project, &d.GitURL, &d.Environment, &d.DeployedAt, &d.Version, &d.DeploymentURL, &d.VersionCreatedAt); err != nil {
			deploymentError(w, err)
			return
		}
		if len(projects) == 0 || projects[len(projects)-1].Project != d.Project {
			projects = append(projects, DeploymentProject{Project: d.Project, Environments: []Deployment{}})
		}
		index := len(projects) - 1
		projects[index].Environments = append(projects[index].Environments, d)
	}
	if err := rows.Err(); err != nil {
		deploymentError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Projects []DeploymentProject `json:"projects"`
	}{projects})
}

func deploymentsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	project := q.Get("project")
	if q.Has("project") && strings.TrimSpace(project) == "" {
		http.Error(w, "project must not be empty when supplied", http.StatusBadRequest)
		return
	}
	limit, offset := 50, 0
	for name, target := range map[string]*int{"limit": &limit, "offset": &offset} {
		if value := q.Get(name); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 0 || (name == "limit" && (parsed == 0 || parsed > 200)) {
				http.Error(w, "limit must be 1–200 and offset must be non-negative", http.StatusBadRequest)
				return
			}
			*target = parsed
		}
	}
	where := " WHERE 1 = 1"
	args := []interface{}{}
	if project != "" {
		where += " AND project = ?"
		args = append(args, project)
	}
	if environment := q.Get("environment"); environment != "" {
		where += " AND environment = ?"
		args = append(args, environment)
	}
	// Keep the count, page, and filter options consistent while reports arrive.
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		deploymentError(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	var total int
	if err := tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM deployments"+where, args...).Scan(&total); err != nil {
		deploymentError(w, err)
		return
	}
	rows, err := tx.QueryContext(r.Context(), "SELECT id, project, git_url, environment, deployed_at, version, deployment_url, version_created_at FROM deployments"+where+
		" ORDER BY deployed_at DESC, id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		deploymentError(w, err)
		return
	}
	deployments := []Deployment{}
	for rows.Next() {
		var d Deployment
		if err := rows.Scan(&d.ID, &d.Project, &d.GitURL, &d.Environment, &d.DeployedAt, &d.Version, &d.DeploymentURL, &d.VersionCreatedAt); err != nil {
			_ = rows.Close()
			deploymentError(w, err)
			return
		}
		deployments = append(deployments, d)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		deploymentError(w, err)
		return
	}
	// Return all relevant environments even when the history is filtered or paginated.
	environmentQuery := "SELECT DISTINCT environment FROM deployments"
	environmentArgs := []interface{}{}
	if project != "" {
		environmentQuery += " WHERE project = ?"
		environmentArgs = append(environmentArgs, project)
	}
	rows, err = tx.QueryContext(r.Context(), environmentQuery+" ORDER BY environment", environmentArgs...)
	if err != nil {
		deploymentError(w, err)
		return
	}
	defer func() { _ = rows.Close() }()
	environments := []string{}
	for rows.Next() {
		var environment string
		if err := rows.Scan(&environment); err != nil {
			deploymentError(w, err)
			return
		}
		environments = append(environments, environment)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		deploymentError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		deploymentError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Deployments  []Deployment `json:"deployments"`
		Total        int          `json:"total"`
		Environments []string     `json:"environments"`
	}{deployments, total, environments})
}

func deploymentError(w http.ResponseWriter, err error) {
	log.Printf("deployment request failed: %v", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func validDeploymentURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Hostname() != "" && parsed.User == nil
}
