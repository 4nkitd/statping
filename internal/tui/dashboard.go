package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ankityadav/statping/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	dashTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

	graphUpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	graphDownStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	metricLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	metricValueStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255"))

	uptimeGoodStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	uptimeBadStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	responseTimeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")).
			Padding(0, 1).
			MarginBottom(1)

	sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
)

type DashboardModel struct {
	db            *storage.Database
	monitors      []storage.Monitor
	checkResults  map[uint][]storage.CheckResult
	width         int
	height        int
	selectedIndex int
	lastUpdate    time.Time
}

type dashTickMsg time.Time

func NewDashboard(db *storage.Database) DashboardModel {
	m := DashboardModel{
		db:           db,
		checkResults: make(map[uint][]storage.CheckResult),
	}
	m.loadData()
	return m
}

func (m *DashboardModel) loadData() {
	monitors, err := m.db.ListMonitors()
	if err != nil {
		return
	}
	m.monitors = monitors

	for _, mon := range monitors {
		results, err := m.db.GetRecentCheckResults(mon.ID, 60)
		if err == nil {
			m.checkResults[mon.ID] = results
		}
	}
	m.lastUpdate = time.Now()
}

func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		dashTickCmd(),
	)
}

func dashTickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return dashTickMsg(t)
	})
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "j", "down":
			if m.selectedIndex < len(m.monitors)-1 {
				m.selectedIndex++
			}
		case "k", "up":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
		case "r":
			m.loadData()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case dashTickMsg:
		m.loadData()
		return m, dashTickCmd()
	}

	return m, nil
}

func (m DashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header
	header := headerStyle.Width(m.width - 2).Render(
		fmt.Sprintf("📊 Statping Dashboard • %d monitors • Updated: %s",
			len(m.monitors),
			m.lastUpdate.Format("15:04:05")))
	b.WriteString(header)
	b.WriteString("\n\n")

	if len(m.monitors) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(
			"No monitors configured. Use 'statping add <url>' to add one."))
		return b.String()
	}

	// Summary cards
	upCount, downCount, unknownCount := m.countStatus()
	summaryCards := m.renderSummaryCards(upCount, downCount, unknownCount)
	b.WriteString(summaryCards)
	b.WriteString("\n\n")

	// Monitor cards with graphs
	for i, mon := range m.monitors {
		selected := i == m.selectedIndex
		card := m.renderMonitorCard(mon, selected)
		b.WriteString(card)
		b.WriteString("\n")
	}

	// Help
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		"j/k: navigate • r: refresh • q: quit")
	b.WriteString("\n")
	b.WriteString(help)

	return b.String()
}

func (m DashboardModel) countStatus() (up, down, unknown int) {
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

func (m DashboardModel) renderSummaryCards(up, down, unknown int) string {
	upCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("42")).
		Padding(0, 2).
		Render(fmt.Sprintf("%s\n%s",
			uptimeGoodStyle.Render(fmt.Sprintf("✓ %d UP", up)),
			metricLabelStyle.Render("Healthy")))

	downCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(0, 2).
		Render(fmt.Sprintf("%s\n%s",
			uptimeBadStyle.Render(fmt.Sprintf("✗ %d DOWN", down)),
			metricLabelStyle.Render("Issues")))

	unknownCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("244")).
		Padding(0, 2).
		Render(fmt.Sprintf("%s\n%s",
			metricValueStyle.Render(fmt.Sprintf("? %d UNKNOWN", unknown)),
			metricLabelStyle.Render("Pending")))

	return lipgloss.JoinHorizontal(lipgloss.Top, upCard, "  ", downCard, "  ", unknownCard)
}

func (m DashboardModel) renderMonitorCard(mon storage.Monitor, selected bool) string {
	results := m.checkResults[mon.ID]

	// Calculate metrics
	var avgResponseTime, minResponseTime, maxResponseTime int64
	var successCount int
	if len(results) > 0 {
		minResponseTime = math.MaxInt64
		for _, r := range results {
			if r.Success {
				successCount++
				avgResponseTime += r.ResponseTime
				if r.ResponseTime < minResponseTime {
					minResponseTime = r.ResponseTime
				}
				if r.ResponseTime > maxResponseTime {
					maxResponseTime = r.ResponseTime
				}
			}
		}
		if successCount > 0 {
			avgResponseTime /= int64(successCount)
		}
		if minResponseTime == math.MaxInt64 {
			minResponseTime = 0
		}
	}

	uptime := float64(0)
	if len(results) > 0 {
		uptime = float64(successCount) / float64(len(results)) * 100
	}

	// Build card content
	var content strings.Builder

	// Name and Status row
	statusIcon := "?"
	statusStyle := metricLabelStyle
	switch mon.CurrentStatus {
	case "up":
		statusIcon = "●"
		statusStyle = uptimeGoodStyle
	case "down":
		statusIcon = "●"
		statusStyle = uptimeBadStyle
	}

	nameRow := fmt.Sprintf("%s %s  %s",
		statusStyle.Render(statusIcon),
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Render(mon.Name),
		metricLabelStyle.Render(truncateURL(mon.URL, 40)))
	content.WriteString(nameRow)
	content.WriteString("\n\n")

	// Response time graph (sparkline)
	graph := m.renderSparkline(results, 50)
	content.WriteString(metricLabelStyle.Render("Response Time (last 60 checks):"))
	content.WriteString("\n")
	content.WriteString(graph)
	content.WriteString("\n\n")

	// Metrics row
	metricsRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderMetric("Uptime", fmt.Sprintf("%.1f%%", uptime), uptime >= 99),
		"   ",
		m.renderMetric("Avg", fmt.Sprintf("%dms", avgResponseTime), true),
		"   ",
		m.renderMetric("Min", fmt.Sprintf("%dms", minResponseTime), true),
		"   ",
		m.renderMetric("Max", fmt.Sprintf("%dms", maxResponseTime), maxResponseTime < 1000),
		"   ",
		m.renderMetric("Checks", fmt.Sprintf("%d", len(results)), true),
	)
	content.WriteString(metricsRow)

	// Last check info
	if mon.LastCheckAt != nil {
		content.WriteString("\n\n")
		lastCheck := fmt.Sprintf("Last check: %s ago", formatTimeAgo(*mon.LastCheckAt))
		content.WriteString(metricLabelStyle.Render(lastCheck))
	}

	// Card border color based on status
	borderColor := lipgloss.Color("240")
	if mon.CurrentStatus == "up" {
		borderColor = lipgloss.Color("42")
	} else if mon.CurrentStatus == "down" {
		borderColor = lipgloss.Color("196")
	}

	cardStyleWithStatus := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(m.width - 4)

	if selected {
		cardStyleWithStatus = cardStyleWithStatus.
			BorderForeground(lipgloss.Color("170")).
			BorderStyle(lipgloss.DoubleBorder())
	}

	return cardStyleWithStatus.Render(content.String())
}

func (m DashboardModel) renderSparkline(results []storage.CheckResult, width int) string {
	if len(results) == 0 {
		return metricLabelStyle.Render("No data yet")
	}

	// Reverse to show oldest to newest (left to right)
	reversed := make([]storage.CheckResult, len(results))
	for i, r := range results {
		reversed[len(results)-1-i] = r
	}

	// Find min/max for scaling
	var maxTime int64 = 1
	for _, r := range reversed {
		if r.ResponseTime > maxTime {
			maxTime = r.ResponseTime
		}
	}

	// Build sparkline
	var spark strings.Builder
	displayCount := width
	if len(reversed) < displayCount {
		displayCount = len(reversed)
	}

	// Start from the end to show most recent
	startIdx := 0
	if len(reversed) > displayCount {
		startIdx = len(reversed) - displayCount
	}

	for i := startIdx; i < len(reversed); i++ {
		r := reversed[i]
		if !r.Success {
			spark.WriteString(graphDownStyle.Render("▄"))
			continue
		}

		// Scale response time to spark block
		normalized := float64(r.ResponseTime) / float64(maxTime)
		blockIdx := int(normalized * float64(len(sparkBlocks)-1))
		if blockIdx >= len(sparkBlocks) {
			blockIdx = len(sparkBlocks) - 1
		}
		if blockIdx < 0 {
			blockIdx = 0
		}

		// Color based on response time
		block := string(sparkBlocks[blockIdx])
		if r.ResponseTime < 200 {
			spark.WriteString(graphUpStyle.Render(block))
		} else if r.ResponseTime < 500 {
			spark.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render(block))
		} else {
			spark.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render(block))
		}
	}

	// Add scale indicator
	scale := fmt.Sprintf(" (0-%dms)", maxTime)
	return spark.String() + metricLabelStyle.Render(scale)
}

func (m DashboardModel) renderMetric(label, value string, good bool) string {
	valueStyle := metricValueStyle
	if !good {
		valueStyle = uptimeBadStyle
	}
	return fmt.Sprintf("%s\n%s",
		valueStyle.Render(value),
		metricLabelStyle.Render(label))
}

func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}

func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
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
