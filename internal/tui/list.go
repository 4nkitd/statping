package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ankityadav/statping/internal/storage"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	statusUpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	statusDownStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	statusUnknownStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))
)

type listModel struct {
	db       *storage.Database
	table    table.Model
	monitors []storage.Monitor
}

func newListModel(db *storage.Database) listModel {
	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Name", Width: 20},
		{Title: "URL", Width: 40},
		{Title: "Status", Width: 10},
		{Title: "Last Check", Width: 20},
		{Title: "Enabled", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	lm := listModel{
		db:    db,
		table: t,
	}
	lm.loadMonitors()
	return lm
}

func (m *listModel) Init() tea.Cmd {
	return nil
}

func (m *listModel) loadMonitors() {
	monitors, err := m.db.ListMonitors()
	if err != nil {
		return
	}
	m.monitors = monitors

	rows := []table.Row{}
	for _, mon := range monitors {
		status := m.formatStatus(mon.CurrentStatus)
		lastCheck := "Never"
		if mon.LastCheckAt != nil {
			lastCheck = formatTime(*mon.LastCheckAt)
		}
		enabled := "No"
		if mon.Enabled {
			enabled = "Yes"
		}

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", mon.ID),
			mon.Name,
			mon.URL,
			status,
			lastCheck,
			enabled,
		})
	}
	m.table.SetRows(rows)
}

func (m *listModel) formatStatus(status string) string {
	switch status {
	case "up":
		return "✓ UP"
	case "down":
		return "✗ DOWN"
	default:
		return "? UNKNOWN"
	}
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "a":
			return m, addMonitor()
		case "e":
			if len(m.monitors) > 0 && m.table.Cursor() < len(m.monitors) {
				return m, editMonitor(&m.monitors[m.table.Cursor()])
			}
		case "d":
			if len(m.monitors) > 0 && m.table.Cursor() < len(m.monitors) {
				monitor := &m.monitors[m.table.Cursor()]
				m.db.DeleteMonitor(monitor.ID)
				m.loadMonitors()
				return m, nil
			}
		case "t":
			if len(m.monitors) > 0 && m.table.Cursor() < len(m.monitors) {
				monitor := &m.monitors[m.table.Cursor()]
				m.db.ToggleMonitor(monitor.ID, !monitor.Enabled)
				m.loadMonitors()
				return m, nil
			}
		case "enter":
			if len(m.monitors) > 0 && m.table.Cursor() < len(m.monitors) {
				return m, monitorSelected(&m.monitors[m.table.Cursor()])
			}
		case "r":
			m.loadMonitors()
			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📊 Statping - Website Monitor"))
	b.WriteString("\n\n")
	b.WriteString(m.table.View())
	b.WriteString("\n\n")

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		"a: add • e: edit • d: delete • t: toggle • enter: details • r: refresh • q: quit",
	)
	b.WriteString(help)

	return b.String()
}

func formatTime(t time.Time) string {
	return t.Format("Jan 02 15:04:05")
}
