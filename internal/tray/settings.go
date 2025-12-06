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
	mux.HandleFunc("/api/monitors", s.handleMonitors)
	mux.HandleFunc("/api/monitor/add", s.handleAddMonitor)
	mux.HandleFunc("/api/monitor/delete", s.handleDeleteMonitor)
	mux.HandleFunc("/api/monitor/toggle", s.handleToggleMonitor)
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
