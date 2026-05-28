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

type colKey int

const (
	colID colKey = iota
	colName
	colURL
	colStatus
	colUptime
	colLast
	colOn
)

type colSpec struct {
	key   colKey
	title string
	width int
	flex  bool
}

type listModel struct {
	db       *storage.Database
	table    table.Model
	monitors []storage.Monitor
	cols     []colSpec
	width    int
}

// cellPad is the padding bubbles table adds on each side of every column.
const cellPad = 2

// listLayout returns a set of columns sized to fit the given terminal width,
// dropping less-important columns on narrow terminals. Flexible columns absorb
// the remaining horizontal space.
func listLayout(width int) []colSpec {
	if width <= 0 {
		width = 100
	}

	var cols []colSpec
	switch {
	case width >= 100:
		cols = []colSpec{
			{key: colID, title: "ID", width: 4},
			{key: colName, title: "Name", flex: true},
			{key: colURL, title: "URL", flex: true},
			{key: colStatus, title: "Status", width: 10},
			{key: colUptime, title: "Uptime", width: 8},
			{key: colLast, title: "Last Check", width: 14},
			{key: colOn, title: "On", width: 3},
		}
	case width >= 78:
		cols = []colSpec{
			{key: colID, title: "ID", width: 4},
			{key: colName, title: "Name", flex: true},
			{key: colURL, title: "URL", flex: true},
			{key: colStatus, title: "Status", width: 10},
			{key: colUptime, title: "Uptime", width: 8},
			{key: colOn, title: "On", width: 3},
		}
	case width >= 56:
		cols = []colSpec{
			{key: colID, title: "ID", width: 4},
			{key: colName, title: "Name", flex: true},
			{key: colStatus, title: "Status", width: 10},
			{key: colUptime, title: "Uptime", width: 7},
			{key: colOn, title: "On", width: 3},
		}
	default:
		cols = []colSpec{
			{key: colID, title: "ID", width: 4},
			{key: colName, title: "Name", flex: true},
			{key: colStatus, title: "Status", width: 9},
		}
	}

	budget := width - len(cols)*cellPad
	fixedTotal, flexCount := 0, 0
	for _, c := range cols {
		if c.flex {
			flexCount++
		} else {
			fixedTotal += c.width
		}
	}

	if flexCount > 0 {
		flexBudget := budget - fixedTotal
		if flexBudget < flexCount*6 {
			flexBudget = flexCount * 6
		}
		per := flexBudget / flexCount
		remaining := flexBudget
		seen := 0
		for i := range cols {
			if !cols[i].flex {
				continue
			}
			seen++
			if seen == flexCount {
				cols[i].width = remaining
			} else {
				cols[i].width = per
				remaining -= per
			}
			if cols[i].width < 6 {
				cols[i].width = 6
			}
		}
	}

	return cols
}

func toTableColumns(cols []colSpec) []table.Column {
	out := make([]table.Column, len(cols))
	for i, c := range cols {
		out[i] = table.Column{Title: c.title, Width: c.width}
	}
	return out
}

func newListModel(db *storage.Database) listModel {
	cols := listLayout(0)
	t := table.New(
		table.WithColumns(toTableColumns(cols)),
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
		cols:  cols,
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

	since := time.Now().Add(-24 * time.Hour)
	rows := []table.Row{}
	for _, mon := range monitors {
		uptime := "—"
		total, successful, _, err := m.db.GetCheckResultStats(mon.ID, since)
		if err == nil && total > 0 {
			uptime = fmt.Sprintf("%.1f%%", float64(successful)/float64(total)*100)
		}

		row := make(table.Row, len(m.cols))
		for i, c := range m.cols {
			row[i] = m.cellFor(mon, c.key, uptime)
		}
		rows = append(rows, row)
	}
	m.table.SetRows(rows)
}

func (m *listModel) cellFor(mon storage.Monitor, key colKey, uptime string) string {
	switch key {
	case colID:
		return fmt.Sprintf("%d", mon.ID)
	case colName:
		return mon.Name
	case colURL:
		return mon.URL
	case colStatus:
		return m.formatStatus(mon.CurrentStatus)
	case colUptime:
		return uptime
	case colLast:
		if mon.LastCheckAt != nil {
			return formatTimeAgo(*mon.LastCheckAt) + " ago"
		}
		return "Never"
	case colOn:
		if mon.Enabled {
			return "✓"
		}
		return "✗"
	default:
		return ""
	}
}

func (m *listModel) setSize(width, height int) {
	m.width = width
	m.cols = listLayout(width)

	// Clear rows before swapping columns: the table re-renders on SetColumns and
	// would panic if existing rows have more cells than the new column count.
	m.table.SetRows(nil)
	m.table.SetColumns(toTableColumns(m.cols))

	h := height - 7
	if h < 3 {
		h = 3
	}
	m.table.SetHeight(h)

	m.loadMonitors()
}

func (m *listModel) countStatus() (up, down, unknown int) {
	for _, mon := range m.monitors {
		switch mon.CurrentStatus {
		case "up":
			up++
		case "down":
			down++
		default:
			unknown++
		}
	}
	return
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
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
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

	up, down, unknown := m.countStatus()
	summary := lipgloss.JoinHorizontal(lipgloss.Top,
		statusUpStyle.Render(fmt.Sprintf("● %d up", up)),
		"  ",
		statusDownStyle.Render(fmt.Sprintf("● %d down", down)),
		"  ",
		statusUnknownStyle.Render(fmt.Sprintf("○ %d unknown", unknown)),
	)

	b.WriteString(titleStyle.Render("📊 Statping"))
	b.WriteString("\n")
	b.WriteString(summary)
	b.WriteString("\n\n")

	if len(m.monitors) == 0 {
		empty := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true).
			Render("No monitors yet. Press 'a' to add one.")
		b.WriteString(empty)
		b.WriteString("\n\n")
	} else {
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(m.helpText()))

	return b.String()
}

func (m listModel) helpText() string {
	full := "a: add • e: edit • d: delete • t: toggle • enter: details • r: refresh • q: quit"
	if m.width == 0 || lipgloss.Width(full) <= m.width {
		return full
	}
	return "a add • e edit • d del • t toggle • r refresh • q quit"
}

func formatTime(t time.Time) string {
	return t.Format("Jan 02 15:04:05")
}
