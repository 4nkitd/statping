package checker

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ankityadav/statping/internal/notifier"
	"github.com/ankityadav/statping/internal/storage"
)

func newTestChecker(t *testing.T) *Checker {
	t.Helper()
	db, err := storage.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	n := notifier.New()
	n.SetEnabled(false)
	return New(db, n)
}

func setProbe(t *testing.T, online bool) {
	t.Helper()
	prev := probeOnline
	probeOnline = func(time.Duration) bool { return online }
	t.Cleanup(func() { probeOnline = prev })
}

func failingResult(id uint) *storage.CheckResult {
	return &storage.CheckResult{
		MonitorID:    id,
		Success:      false,
		ErrorMessage: "dial tcp: connection refused",
		CreatedAt:    time.Now(),
	}
}

func TestFailureSuppressedWhenOffline(t *testing.T) {
	c := newTestChecker(t)
	setProbe(t, false) // system is offline

	m := &storage.Monitor{Name: "Site", URL: "https://x.test", MaxFailures: 2, CurrentStatus: "up", Enabled: true}
	if err := c.db.CreateMonitor(m); err != nil {
		t.Fatalf("create monitor: %v", err)
	}

	for i := 0; i < 5; i++ {
		c.recordFailure(m, failingResult(m.ID))
	}

	if m.ConsecutiveFails != 0 {
		t.Errorf("expected no failures counted while offline, got %d", m.ConsecutiveFails)
	}
	if m.CurrentStatus != "up" {
		t.Errorf("expected status to stay 'up' while offline, got %q", m.CurrentStatus)
	}
	if c.Online() {
		t.Error("expected checker to detect it is offline")
	}

	results, _ := c.db.GetRecentCheckResults(m.ID, 10)
	if len(results) != 0 {
		t.Errorf("expected no check results recorded while offline, got %d", len(results))
	}
	if inc, _ := c.db.GetActiveIncident(m.ID); inc != nil {
		t.Error("expected no incident opened while offline")
	}
}

func TestFailureMarksDownWhenOnline(t *testing.T) {
	c := newTestChecker(t)
	setProbe(t, true) // system is online

	m := &storage.Monitor{Name: "Site", URL: "https://x.test", MaxFailures: 3, CurrentStatus: "up", Enabled: true}
	if err := c.db.CreateMonitor(m); err != nil {
		t.Fatalf("create monitor: %v", err)
	}

	for i := 0; i < 3; i++ {
		c.recordFailure(m, failingResult(m.ID))
	}

	if m.CurrentStatus != "down" {
		t.Errorf("expected status 'down' after %d failures, got %q", m.MaxFailures, m.CurrentStatus)
	}
	if inc, _ := c.db.GetActiveIncident(m.ID); inc == nil {
		t.Error("expected an active incident to be opened")
	}
}

func TestSuccessClearsOfflineFlag(t *testing.T) {
	c := newTestChecker(t)
	c.setOnline(false)

	m := &storage.Monitor{Name: "Site", URL: "https://x.test", CurrentStatus: "down", Enabled: true}
	if err := c.db.CreateMonitor(m); err != nil {
		t.Fatalf("create monitor: %v", err)
	}

	c.recordSuccess(m, &storage.CheckResult{MonitorID: m.ID, Success: true, StatusCode: 200, ResponseTime: 50, CreatedAt: time.Now()})

	if !c.Online() {
		t.Error("expected a successful check to mark the system online")
	}
	if m.CurrentStatus != "up" {
		t.Errorf("expected status 'up' after success, got %q", m.CurrentStatus)
	}
}
