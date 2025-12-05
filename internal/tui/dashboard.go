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

// Dashboard color palette
var (
	dColorGreen   = lipgloss.Color("#04B575")
	dColorRed     = lipgloss.Color("#FF4D4D")
	dColorYellow  = lipgloss.Color("#FFCC00")
	dColorOrange  = lipgloss.Color("#FF8C00")
	dColorPurple  = lipgloss.Color("#BD93F9")
	dColorGray    = lipgloss.Color("#6C7086")
	dColorDimGray = lipgloss.Color("#45475A")
	dColorWhite   = lipgloss.Color("#CDD6F4")
)

// Dashboard styles
var (
	dHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(dColorWhite).
			Background(dColorPurple).
			Padding(0, 2)

	dSubtitleStyle = lipgloss.NewStyle().
			Foreground(dColorGray)

	dCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dColorDimGray).
			Padding(1, 2).
			MarginBottom(1)

	dCardSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(dColorPurple).
				Padding(1, 2).
				MarginBottom(1)

	dStatusUpStyle = lipgloss.NewStyle().
			Foreground(dColorGreen).
			Bold(true)

	dStatusDownStyle = lipgloss.NewStyle().
				Foreground(dColorRed).
				Bold(true)

	dStatusUnknownStyle = lipgloss.NewStyle().
				Foreground(dColorGray).
				Bold(true)

	dMetricLabelStyle = lipgloss.NewStyle().
				Foreground(dColorGray)

	dMetricValueStyle = lipgloss.NewStyle().
				Foreground(dColorWhite).
				Bold(true)

	dMetricGoodStyle = lipgloss.NewStyle().
				Foreground(dColorGreen).
				Bold(true)

	dMetricBadStyle = lipgloss.NewStyle().
			Foreground(dColorRed).
			Bold(true)

	dMetricWarnStyle = lipgloss.NewStyle().
				Foreground(dColorYellow).
				Bold(true)

	dMonitorNameStyle = lipgloss.NewStyle().
				Foreground(dColorWhite).
				Bold(true)

	dUrlStyle = lipgloss.NewStyle().
			Foreground(dColorGray)

	dGraphGreenStyle = lipgloss.NewStyle().
				Foreground(dColorGreen)

	dGraphYellowStyle = lipgloss.NewStyle().
				Foreground(dColorYellow)

	dGraphOrangeStyle = lipgloss.NewStyle().
				Foreground(dColorOrange)

	dGraphRedStyle = lipgloss.NewStyle().
			Foreground(dColorRed)

	dHelpStyle = lipgloss.NewStyle().
			Foreground(dColorDimGray)

	dHelpKeyStyle = lipgloss.NewStyle().
			Foreground(dColorPurple).
			Bold(true)

	dSparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
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

	// Header with gradient-like effect
	headerText := " 📊 STATPING DASHBOARD "
	header := dHeaderStyle.Render(headerText)
	statsText := dSubtitleStyle.Render(fmt.Sprintf("  %d monitors • Updated %s", len(m.monitors), m.lastUpdate.Format("15:04:05")))
	b.WriteString(header + statsText)
	b.WriteString("\n\n")

	if len(m.monitors) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(dColorGray).
			Italic(true).
			Render("  No monitors configured. Use 'statping add <url>' to add one.")
		b.WriteString(emptyMsg)
		return b.String()
	}

	// Summary cards with better styling
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

	// Help bar with styled keys
	helpText := fmt.Sprintf("%s navigate • %s refresh • %s quit",
		dHelpKeyStyle.Render("↑↓"),
		dHelpKeyStyle.Render("r"),
		dHelpKeyStyle.Render("q"))
	b.WriteString(dHelpStyle.Render(helpText))

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
		BorderForeground(dColorGreen).
		Padding(0, 3).
		Render(fmt.Sprintf("%s\n%s",
			dStatusUpStyle.Render(fmt.Sprintf("✓ %d UP", up)),
			dMetricLabelStyle.Render("Healthy")))

	downCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dColorRed).
		Padding(0, 3).
		Render(fmt.Sprintf("%s\n%s",
			dStatusDownStyle.Render(fmt.Sprintf("✗ %d DOWN", down)),
			dMetricLabelStyle.Render("Issues")))

	unknownCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dColorGray).
		Padding(0, 3).
		Render(fmt.Sprintf("%s\n%s",
			dStatusUnknownStyle.Render(fmt.Sprintf("? %d UNKNOWN", unknown)),
			dMetricLabelStyle.Render("Pending")))

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

	// Status indicator and name
	var statusIcon string
	var statusStyle lipgloss.Style
	switch mon.CurrentStatus {
	case "up":
		statusIcon = "●"
		statusStyle = dStatusUpStyle
	case "down":
		statusIcon = "●"
		statusStyle = dStatusDownStyle
	default:
		statusIcon = "○"
		statusStyle = dStatusUnknownStyle
	}

	// Header row with status, name, and URL
	nameRow := fmt.Sprintf("%s %s  %s",
		statusStyle.Render(statusIcon),
		dMonitorNameStyle.Render(mon.Name),
		dUrlStyle.Render(truncateURL(mon.URL, 45)))
	content.WriteString(nameRow)
	content.WriteString("\n\n")

	// Response time graph label
	content.WriteString(dMetricLabelStyle.Render("Response Time (last 60 checks):"))
	content.WriteString("\n")

	// Sparkline graph
	graph := m.renderSparkline(results, 60)
	content.WriteString(graph)
	content.WriteString("\n\n")

	// Metrics row with better spacing
	metricsRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderMetric("Uptime", fmt.Sprintf("%.1f%%", uptime), uptime >= 99),
		"    ",
		m.renderMetric("Avg", fmt.Sprintf("%dms", avgResponseTime), avgResponseTime < 500),
		"    ",
		m.renderMetric("Min", fmt.Sprintf("%dms", minResponseTime), true),
		"    ",
		m.renderMetric("Max", fmt.Sprintf("%dms", maxResponseTime), maxResponseTime < 1000),
		"    ",
		m.renderMetric("Checks", fmt.Sprintf("%d", len(results)), true),
	)
	content.WriteString(metricsRow)

	// Last check info
	if mon.LastCheckAt != nil {
		content.WriteString("\n\n")
		lastCheck := fmt.Sprintf("Last check: %s ago", formatTimeAgo(*mon.LastCheckAt))
		content.WriteString(dMetricLabelStyle.Render(lastCheck))
	}

	// Card styling based on status and selection
	var cardStyleFinal lipgloss.Style
	if selected {
		cardStyleFinal = dCardSelectedStyle.
			Width(m.width - 4).
			BorderForeground(dColorPurple)
	} else {
		borderColor := dColorDimGray
		if mon.CurrentStatus == "up" {
			borderColor = dColorGreen
		} else if mon.CurrentStatus == "down" {
			borderColor = dColorRed
		}
		cardStyleFinal = dCardStyle.
			Width(m.width - 4).
			BorderForeground(borderColor)
	}

	return cardStyleFinal.Render(content.String())
}

func (m DashboardModel) renderSparkline(results []storage.CheckResult, width int) string {
	if len(results) == 0 {
		return dMetricLabelStyle.Render("No data yet")
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
			spark.WriteString(dGraphRedStyle.Render("▄"))
			continue
		}

		// Scale response time to spark block
		normalized := float64(r.ResponseTime) / float64(maxTime)
		blockIdx := int(normalized * float64(len(dSparkBlocks)-1))
		if blockIdx >= len(dSparkBlocks) {
			blockIdx = len(dSparkBlocks) - 1
		}
		if blockIdx < 0 {
			blockIdx = 0
		}

		// Color based on response time
		block := string(dSparkBlocks[blockIdx])
		if r.ResponseTime < 200 {
			spark.WriteString(dGraphGreenStyle.Render(block))
		} else if r.ResponseTime < 500 {
			spark.WriteString(dGraphYellowStyle.Render(block))
		} else {
			spark.WriteString(dGraphOrangeStyle.Render(block))
		}
	}

	// Add scale indicator
	scale := fmt.Sprintf(" (0-%dms)", maxTime)
	return spark.String() + dMetricLabelStyle.Render(scale)
}

func (m DashboardModel) renderMetric(label, value string, good bool) string {
	var valueStyle lipgloss.Style
	if good {
		valueStyle = dMetricValueStyle
	} else {
		valueStyle = dMetricWarnStyle
	}
	return fmt.Sprintf("%s\n%s",
		valueStyle.Render(value),
		dMetricLabelStyle.Render(label))
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
