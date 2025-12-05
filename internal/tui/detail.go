package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ankityadav/statping/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type detailModel struct {
	db           *storage.Database
	monitor      *storage.Monitor
	checkResults []storage.CheckResult
	incidents    []storage.Incident
}

func newDetailModel(db *storage.Database) detailModel {
	return detailModel{
		db: db,
	}
}

func (m *detailModel) setMonitor(monitor *storage.Monitor) {
	m.monitor = monitor
	m.refresh()
}

func (m *detailModel) refresh() {
	if m.monitor == nil {
		return
	}

	mon, err := m.db.GetMonitor(m.monitor.ID)
	if err == nil {
		m.monitor = mon
	}

	results, err := m.db.GetRecentCheckResults(m.monitor.ID, 10)
	if err == nil {
		m.checkResults = results
	}

	incidents, err := m.db.GetRecentIncidents(m.monitor.ID, 5)
	if err == nil {
		m.incidents = incidents
	}
}

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, backToList()
		case "e":
			return m, editMonitor(m.monitor)
		}
	}
	return m, nil
}

func (m detailModel) View() string {
	if m.monitor == nil {
		return "No monitor selected"
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("Monitor Details: %s", m.monitor.Name)))
	b.WriteString("\n\n")

	infoStyle := lipgloss.NewStyle().Bold(true)
	b.WriteString(infoStyle.Render("URL: "))
	b.WriteString(m.monitor.URL)
	b.WriteString("\n")

	b.WriteString(infoStyle.Render("Status: "))
	status := m.formatStatus(m.monitor.CurrentStatus)
	b.WriteString(status)
	b.WriteString("\n")

	b.WriteString(infoStyle.Render("Check Interval: "))
	b.WriteString(fmt.Sprintf("%d seconds", m.monitor.CheckInterval))
	b.WriteString("\n")

	b.WriteString(infoStyle.Render("Timeout: "))
	b.WriteString(fmt.Sprintf("%d seconds", m.monitor.Timeout))
	b.WriteString("\n")

	b.WriteString(infoStyle.Render("Expected Codes: "))
	b.WriteString(m.monitor.ExpectedCodes)
	b.WriteString("\n")

	if m.monitor.Keywords != "" {
		b.WriteString(infoStyle.Render("Keywords: "))
		b.WriteString(m.monitor.Keywords)
		b.WriteString("\n")
	}

	b.WriteString(infoStyle.Render("Enabled: "))
	if m.monitor.Enabled {
		b.WriteString("Yes")
	} else {
		b.WriteString("No")
	}
	b.WriteString("\n")

	if m.monitor.LastCheckAt != nil {
		b.WriteString(infoStyle.Render("Last Check: "))
		b.WriteString(m.monitor.LastCheckAt.Format("2006-01-02 15:04:05"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Statistics (Last 24h)"))
	b.WriteString("\n")

	since := time.Now().Add(-24 * time.Hour)
	total, successful, avgResponseTime, err := m.db.GetCheckResultStats(m.monitor.ID, since)
	if err == nil && total > 0 {
		uptime := float64(successful) / float64(total) * 100
		b.WriteString(fmt.Sprintf("Uptime: %.2f%% (%d/%d checks)\n", uptime, successful, total))
		b.WriteString(fmt.Sprintf("Avg Response Time: %.0fms\n", avgResponseTime))
	} else {
		b.WriteString("No data available\n")
	}

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Recent Checks"))
	b.WriteString("\n")

	if len(m.checkResults) > 0 {
		for _, cr := range m.checkResults {
			statusIcon := "✓"
			if !cr.Success {
				statusIcon = "✗"
			}
			timeStr := cr.CreatedAt.Format("15:04:05")
			b.WriteString(fmt.Sprintf("%s %s - ", statusIcon, timeStr))

			if cr.Success {
				b.WriteString(fmt.Sprintf("HTTP %d (%dms)", cr.StatusCode, cr.ResponseTime))
			} else {
				b.WriteString(fmt.Sprintf("Failed: %s", cr.ErrorMessage))
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString("No check results yet\n")
	}

	if len(m.incidents) > 0 {
		b.WriteString("\n")
		b.WriteString(titleStyle.Render("Recent Incidents"))
		b.WriteString("\n")

		for _, inc := range m.incidents {
			b.WriteString(fmt.Sprintf("Started: %s\n", inc.StartedAt.Format("2006-01-02 15:04:05")))
			if inc.ResolvedAt != nil {
				duration := inc.ResolvedAt.Sub(inc.StartedAt)
				b.WriteString(fmt.Sprintf("Resolved: %s (Duration: %s)\n",
					inc.ResolvedAt.Format("2006-01-02 15:04:05"),
					formatDuration(duration)))
			} else {
				duration := time.Since(inc.StartedAt)
				b.WriteString(fmt.Sprintf("Status: ONGOING (Duration: %s)\n", formatDuration(duration)))
			}
			b.WriteString(fmt.Sprintf("Error: %s\n\n", inc.ErrorMessage))
		}
	}

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		"e: edit • esc/q: back to list",
	)
	b.WriteString("\n")
	b.WriteString(help)

	return b.String()
}

func (m detailModel) formatStatus(status string) string {
	switch status {
	case "up":
		return statusUpStyle.Render("✓ UP")
	case "down":
		return statusDownStyle.Render("✗ DOWN")
	default:
		return statusUnknownStyle.Render("? UNKNOWN")
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}
