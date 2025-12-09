package tray

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/ankityadav/statping/internal/storage"
)

//go:embed templates/*
var templatesFS embed.FS

type SettingsServer struct {
	db       *storage.Database
	onUpdate func()
	server   *http.Server
	port     int
	mu       sync.Mutex
}

func NewSettingsWindow(db *storage.Database, onUpdate func()) *SettingsServer {
	return &SettingsServer{
		db:       db,
		onUpdate: onUpdate,
	}
}

func (s *SettingsServer) Show() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return
	}
	s.port = listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/site/", s.handleSiteDetail)
	mux.HandleFunc("/api/monitors", s.handleMonitors)
	mux.HandleFunc("/api/monitor/add", s.handleAddMonitor)
	mux.HandleFunc("/api/monitor/delete", s.handleDeleteMonitor)
	mux.HandleFunc("/api/monitor/toggle", s.handleToggleMonitor)
	mux.HandleFunc("/api/monitor/stats", s.handleMonitorStats)
	mux.HandleFunc("/api/monitor/checks", s.handleMonitorChecks)
	mux.HandleFunc("/api/monitor/incidents", s.handleMonitorIncidents)
	mux.HandleFunc("/static/style.css", s.handleCSS)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler: mux,
	}

	go s.server.ListenAndServe()

	// Open browser
	url := fmt.Sprintf("http://127.0.0.1:%d", s.port)
	openBrowser(url)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

func (s *SettingsServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/index.html"))
	monitors, _ := s.db.ListMonitors()
	tmpl.Execute(w, map[string]interface{}{
		"Monitors": monitors,
		"Port":     s.port,
	})
}

func (s *SettingsServer) handleCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	data, _ := templatesFS.ReadFile("templates/style.css")
	w.Write(data)
}

func (s *SettingsServer) handleMonitors(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.db.ListMonitors()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monitors)
}

func (s *SettingsServer) handleAddMonitor(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Name          string `json:"name"`
		URL           string `json:"url"`
		Interval      int    `json:"interval"`
		Timeout       int    `json:"timeout"`
		ExpectedCodes string `json:"expected_codes"`
		Keywords      string `json:"keywords"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", 400)
		return
	}

	name := req.Name
	if name == "" {
		name = req.URL
	}

	interval := req.Interval
	if interval <= 0 {
		interval = 60
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 10
	}

	codes := req.ExpectedCodes
	if codes == "" {
		codes = "200"
	}

	monitor := &storage.Monitor{
		Name:          name,
		URL:           req.URL,
		CheckInterval: interval,
		Timeout:       timeout,
		ExpectedCodes: codes,
		Keywords:      req.Keywords,
		Enabled:       true,
	}

	if err := s.db.CreateMonitor(monitor); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if s.onUpdate != nil {
		s.onUpdate()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "id": monitor.ID})
}

func (s *SettingsServer) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid ID", 400)
		return
	}

	if err := s.db.DeleteMonitor(uint(id)); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if s.onUpdate != nil {
		s.onUpdate()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *SettingsServer) handleToggleMonitor(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid ID", 400)
		return
	}

	monitor, err := s.db.GetMonitor(uint(id))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	monitor.Enabled = !monitor.Enabled
	if err := s.db.UpdateMonitor(monitor); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if s.onUpdate != nil {
		s.onUpdate()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true, "enabled": monitor.Enabled})
}

func (s *SettingsServer) handleSiteDetail(w http.ResponseWriter, r *http.Request) {
	// Extract ID from /site/123
	path := r.URL.Path
	idStr := path[len("/site/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid ID", 400)
		return
	}

	monitor, err := s.db.GetMonitor(uint(id))
	if err != nil {
		http.Error(w, "Monitor not found", 404)
		return
	}

	tmpl := template.Must(template.ParseFS(templatesFS, "templates/detail.html"))
	tmpl.Execute(w, map[string]interface{}{
		"Monitor": monitor,
		"Port":    s.port,
	})
}

func (s *SettingsServer) handleMonitorStats(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid ID", 400)
		return
	}

	period := r.URL.Query().Get("period")
	var since time.Time
	switch period {
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	default:
		since = time.Now().Add(-24 * time.Hour)
	}

	total, successful, avgResponseTime, err := s.db.GetCheckResultStats(uint(id), since)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	uptime := float64(0)
	if total > 0 {
		uptime = float64(successful) / float64(total) * 100
	}

	// Get incidents count
	incidents, _ := s.db.GetRecentIncidents(uint(id), 100)
	incidentCount := 0
	var totalDowntime time.Duration
	for _, inc := range incidents {
		if inc.StartedAt.After(since) {
			incidentCount++
			if inc.ResolvedAt != nil {
				totalDowntime += inc.ResolvedAt.Sub(inc.StartedAt)
			} else {
				totalDowntime += time.Since(inc.StartedAt)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_checks":      total,
		"successful_checks": successful,
		"failed_checks":     total - successful,
		"uptime":            uptime,
		"avg_response_time": avgResponseTime,
		"incident_count":    incidentCount,
		"total_downtime":    totalDowntime.String(),
		"downtime_minutes":  totalDowntime.Minutes(),
	})
}

func (s *SettingsServer) handleMonitorChecks(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid ID", 400)
		return
	}

	// Get period from query params (default 24h)
	period := r.URL.Query().Get("period")
	var since time.Time
	switch period {
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	default:
		since = time.Now().Add(-24 * time.Hour)
	}

	results, err := s.db.GetCheckResultsSince(uint(id), since)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Convert to JSON-friendly format with timestamps
	type CheckData struct {
		Timestamp    string `json:"timestamp"`
		ResponseTime int64  `json:"response_time"`
		StatusCode   int    `json:"status_code"`
		Success      bool   `json:"success"`
		Error        string `json:"error,omitempty"`
	}

	checks := make([]CheckData, len(results))
	for i, r := range results {
		checks[i] = CheckData{
			Timestamp:    r.CreatedAt.Format(time.RFC3339),
			ResponseTime: r.ResponseTime,
			StatusCode:   r.StatusCode,
			Success:      r.Success,
			Error:        r.ErrorMessage,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checks)
}

func (s *SettingsServer) handleMonitorIncidents(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid ID", 400)
		return
	}

	incidents, err := s.db.GetRecentIncidents(uint(id), 50)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	type IncidentData struct {
		ID         uint    `json:"id"`
		StartedAt  string  `json:"started_at"`
		ResolvedAt *string `json:"resolved_at"`
		Duration   string  `json:"duration"`
		Error      string  `json:"error"`
		Resolved   bool    `json:"resolved"`
	}

	data := make([]IncidentData, len(incidents))
	for i, inc := range incidents {
		var resolvedAt *string
		if inc.ResolvedAt != nil {
			t := inc.ResolvedAt.Format(time.RFC3339)
			resolvedAt = &t
		}

		var duration time.Duration
		if inc.ResolvedAt != nil {
			duration = inc.ResolvedAt.Sub(inc.StartedAt)
		} else {
			duration = time.Since(inc.StartedAt)
		}

		data[i] = IncidentData{
			ID:         inc.ID,
			StartedAt:  inc.StartedAt.Format(time.RFC3339),
			ResolvedAt: resolvedAt,
			Duration:   formatDurationHuman(duration),
			Error:      inc.ErrorMessage,
			Resolved:   inc.ResolvedAt != nil,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func formatDurationHuman(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}
