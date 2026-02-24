package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed frontend/dist
var frontendFS embed.FS

const dbPath = "/data/db.sqlite"

type WebhookPayload struct {
	Project    string `json:"project"`
	Spec       string `json:"spec"`
	Browser    string `json:"browser"`
	Status     string `json:"status"`
	PipelineID string `json:"pipeline_id"`
	JobID      string `json:"job_id"`
	JobURL     string `json:"job_url"`
}

type TestResult struct {
	ID         int64  `json:"id"`
	Project    string `json:"project"`
	Spec       string `json:"spec"`
	Browser    string `json:"browser"`
	Status     string `json:"status"`
	PipelineID string `json:"pipeline_id"`
	JobID      string `json:"job_id"`
	JobURL     string `json:"job_url"`
	CreatedAt  string `json:"created_at"`
}

type ProjectSummary struct {
	Project      string `json:"project"`
	LatestStatus string `json:"latest_status"`
	TotalRuns    int    `json:"total_runs"`
	PassedRuns   int    `json:"passed_runs"`
	FailedRuns   int    `json:"failed_runs"`
	LastRun      string `json:"last_run"`
}

var db *sql.DB

func initDB() {
	// Ensure /data directory exists
	if err := os.MkdirAll("/data", 0755); err != nil {
		log.Fatalf("failed to create /data directory: %v", err)
	}

	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS test_results (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		project     TEXT NOT NULL,
		spec        TEXT NOT NULL,
		browser     TEXT NOT NULL,
		status      TEXT NOT NULL,
		pipeline_id TEXT NOT NULL,
		job_id      TEXT NOT NULL,
		job_url     TEXT NOT NULL,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_project ON test_results(project);
	CREATE INDEX IF NOT EXISTS idx_status ON test_results(status);
	CREATE INDEX IF NOT EXISTS idx_created_at ON test_results(created_at);
	`
	if _, err := db.Exec(createTable); err != nil {
		log.Fatalf("failed to create table: %v", err)
	}
}

func authMiddleware(pathPrefix string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only protect the webhook endpoint
		if r.URL.Path == pathPrefix+"/webhook" {
			token := os.Getenv("API_TOKEN")
			if token != "" {
				authHeader := r.Header.Get("Authorization")
				if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != token {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func normalizePathPrefix(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "/" {
		return ""
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	return strings.TrimSuffix(raw, "/")
}

func serveIndexHTML(w http.ResponseWriter, distFS fs.FS, pathPrefix string, footerText string) {
	data, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		http.Error(w, "index not found", http.StatusNotFound)
		return
	}

	html := string(data)
	scriptParts := []string{}
	if pathPrefix != "" {
		html = strings.ReplaceAll(html, "href=\"/assets/", "href=\""+pathPrefix+"/assets/")
		html = strings.ReplaceAll(html, "src=\"/assets/", "src=\""+pathPrefix+"/assets/")
		html = strings.ReplaceAll(html, "href=\"/favicon.svg\"", "href=\""+pathPrefix+"/favicon.svg\"")
		scriptParts = append(scriptParts, "window.__PATH_PREFIX__="+strconv.Quote(pathPrefix))
	}
	if strings.TrimSpace(footerText) != "" {
		scriptParts = append(scriptParts, "window.__FOOTER_TEXT__="+strconv.Quote(footerText))
	}
	if len(scriptParts) > 0 {
		script := "<script>" + strings.Join(scriptParts, ";") + ";</script>"
		html = strings.Replace(html, "</head>", script+"</head>", 1)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if payload.Project == "" || payload.Status == "" {
		http.Error(w, "project and status are required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(
		`INSERT INTO test_results (project, spec, browser, status, pipeline_id, job_id, job_url) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		payload.Project, payload.Spec, payload.Browser, payload.Status,
		payload.PipelineID, payload.JobID, payload.JobURL,
	)
	if err != nil {
		log.Printf("failed to insert test result: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("failed to encode webhook response: %v", err)
	}
}

func projectsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT
			project,
			status AS latest_status,
			COUNT(*) AS total_runs,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS passed_runs,
			SUM(CASE WHEN status != 'success' THEN 1 ELSE 0 END) AS failed_runs,
			MAX(created_at) AS last_run
		FROM test_results
		GROUP BY project
		ORDER BY last_run DESC
	`)
	if err != nil {
		log.Printf("failed to query projects: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = rows.Close() }()

	summaries := []ProjectSummary{}
	for rows.Next() {
		var s ProjectSummary
		if err := rows.Scan(&s.Project, &s.LatestStatus, &s.TotalRuns, &s.PassedRuns, &s.FailedRuns, &s.LastRun); err != nil {
			continue
		}
		summaries = append(summaries, s)
	}

	// Determine latest status per project from most recent run
	for i, s := range summaries {
		var latestStatus string
		err := db.QueryRow(
			`SELECT status FROM test_results WHERE project = ? ORDER BY created_at DESC LIMIT 1`,
			s.Project,
		).Scan(&latestStatus)
		if err == nil {
			summaries[i].LatestStatus = latestStatus
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summaries); err != nil {
		log.Printf("failed to encode projects response: %v", err)
	}
}

func projectResultsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	project := vars["project"]

	// Optional query params
	q := r.URL.Query()
	statusFilter := q.Get("status")
	browserFilter := q.Get("browser")
	specFilter := q.Get("spec")
	limitStr := q.Get("limit")
	if limitStr == "" {
		limitStr = "100"
	}

	query := `SELECT id, project, spec, browser, status, pipeline_id, job_id, job_url, created_at
	          FROM test_results WHERE project = ?`
	args := []interface{}{project}

	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
	}
	if browserFilter != "" {
		query += " AND browser = ?"
		args = append(args, browserFilter)
	}
	if specFilter != "" {
		query += " AND spec LIKE ?"
		args = append(args, "%"+specFilter+"%")
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limitStr)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("failed to query results: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = rows.Close() }()

	results := []TestResult{}
	for rows.Next() {
		var r TestResult
		if err := rows.Scan(&r.ID, &r.Project, &r.Spec, &r.Browser, &r.Status,
			&r.PipelineID, &r.JobID, &r.JobURL, &r.CreatedAt); err != nil {
			continue
		}
		results = append(results, r)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("failed to encode project results response: %v", err)
	}
}

func allResultsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	statusFilter := q.Get("status")
	browserFilter := q.Get("browser")
	limitStr := q.Get("limit")
	if limitStr == "" {
		limitStr = "200"
	}

	query := `SELECT id, project, spec, browser, status, pipeline_id, job_id, job_url, created_at
	          FROM test_results WHERE 1=1`
	args := []interface{}{}

	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
	}
	if browserFilter != "" {
		query += " AND browser = ?"
		args = append(args, browserFilter)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limitStr)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("failed to query all results: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = rows.Close() }()

	results := []TestResult{}
	for rows.Next() {
		var r TestResult
		if err := rows.Scan(&r.ID, &r.Project, &r.Spec, &r.Browser, &r.Status,
			&r.PipelineID, &r.JobID, &r.JobURL, &r.CreatedAt); err != nil {
			continue
		}
		results = append(results, r)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("failed to encode all results response: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		log.Printf("failed to encode health response: %v", err)
	}
}

func main() {
	initDB()

	pathPrefix := normalizePathPrefix(os.Getenv("PATH_PREFIX"))
	footerText := os.Getenv("FOOTER_TEXT")

	r := mux.NewRouter()
	appRouter := r
	if pathPrefix != "" {
		appRouter = r.PathPrefix(pathPrefix).Subrouter()
	}

	// API routes
	api := appRouter.PathPrefix("/api").Subrouter()
	api.HandleFunc("/projects", projectsHandler).Methods("GET")
	api.HandleFunc("/projects/{project}/results", projectResultsHandler).Methods("GET")
	api.HandleFunc("/results", allResultsHandler).Methods("GET")
	api.HandleFunc("/health", healthHandler).Methods("GET")

	// Webhook
	appRouter.HandleFunc("/webhook", webhookHandler).Methods("POST")

	// Serve Vue SPA - strip the "frontend/dist" prefix from the embedded FS
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Fatalf("failed to create sub filesystem: %v", err)
	}
	fileServer := http.FileServer(http.FS(distFS))

	// All other routes serve the SPA (Vue Router handles client-side routing)
	appRouter.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Try to serve file; if not found, serve index.html for SPA routing
		requestPath := req.URL.Path
		if pathPrefix != "" {
			requestPath = strings.TrimPrefix(requestPath, pathPrefix)
		}
		path := strings.TrimPrefix(requestPath, "/")
		if path == "" {
			serveIndexHTML(w, distFS, pathPrefix, footerText)
			return
		}
		if _, err := fs.Stat(distFS, path); err != nil {
			// File not found - serve index.html for SPA
			serveIndexHTML(w, distFS, pathPrefix, footerText)
			return
		}
		req.URL.Path = requestPath
		fileServer.ServeHTTP(w, req)
	})

	handler := authMiddleware(pathPrefix, r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
