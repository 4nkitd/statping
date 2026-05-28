package tui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ankityadav/statping/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newTestDB(t *testing.T) *storage.Database {
	t.Helper()
	db, err := storage.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

func seedMonitors(t *testing.T, db *storage.Database, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		m := &storage.Monitor{
			Name:          "Site",
			URL:           "https://example.com/" + time.Now().Format("150405.000000000"),
			CheckInterval: 60,
			Timeout:       10,
			Enabled:       true,
			CurrentStatus: "up",
		}
		if err := db.CreateMonitor(m); err != nil {
			t.Fatalf("create monitor: %v", err)
		}
		for j := 0; j < 5; j++ {
			_ = db.CreateCheckResult(&storage.CheckResult{
				MonitorID:    m.ID,
				StatusCode:   200,
				ResponseTime: int64(100 + j*10),
				Success:      true,
				CreatedAt:    time.Now(),
			})
		}
	}
}

func TestDashboardFitsTerminalWithManyMonitors(t *testing.T) {
	db := newTestDB(t)
	seedMonitors(t, db, 50)

	for _, sz := range terminalSizes {
		var m tea.Model = NewDashboard(db)
		m, _ = m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})

		view := m.View()
		if got := lipgloss.Height(view); got > sz.h {
			t.Fatalf("[%dx%d] dashboard rendered %d lines, exceeds terminal height", sz.w, sz.h, got)
		}
		if got := lipgloss.Width(view); got > sz.w {
			t.Fatalf("[%dx%d] dashboard rendered width %d, exceeds terminal width", sz.w, sz.h, got)
		}
	}
}

func TestDashboardUsesMultipleColumnsWhenWide(t *testing.T) {
	db := newTestDB(t)
	seedMonitors(t, db, 8)

	var m tea.Model = NewDashboard(db)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 190, Height: 50})

	dm := m.(DashboardModel)
	if dm.numCols < 2 {
		t.Fatalf("expected multiple columns on a 190-wide terminal, got %d", dm.numCols)
	}

	// Narrow terminal should fall back to a single column.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 70, Height: 50})
	dm = m.(DashboardModel)
	if dm.numCols != 1 {
		t.Fatalf("expected single column on a 70-wide terminal, got %d", dm.numCols)
	}
}

func TestDashboardScrollsToSelection(t *testing.T) {
	db := newTestDB(t)
	seedMonitors(t, db, 50)

	const width, height = 100, 30

	var m tea.Model = NewDashboard(db)
	m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})

	for i := 0; i < 49; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	dm := m.(DashboardModel)
	if dm.selectedIndex != 49 {
		t.Fatalf("expected selectedIndex 49, got %d", dm.selectedIndex)
	}
	if dm.viewport.YOffset == 0 {
		t.Fatal("expected viewport to scroll for the last selection, but YOffset is 0")
	}
	if got := lipgloss.Height(m.View()); got > height {
		t.Fatalf("dashboard rendered %d lines after scrolling, exceeds height %d", got, height)
	}
}
