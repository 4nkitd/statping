package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ankityadav/statping/internal/storage"
	"github.com/charmbracelet/bubbles/viewport"
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
	viewport      viewport.Model
	ready         bool
	numCols       int
}

const (
	cardGap           = 2
	cardMinWidth      = 54 // minimum total card width (incl. border + padding)
	cardMaxCols       = 4
	cardContentHeight = 9 // fixed content height so cards in a row align
)

// gridDims returns how many cards fit per row and the total width of each card
// for the current terminal width.
func (m DashboardModel) gridDims() (cols, cardWidth int) {
	if m.width <= 0 {
		return 1, 0
	}
	cols = (m.width + cardGap) / (cardMinWidth + cardGap)
	if cols < 1 {
		cols = 1
	}
	if cols > cardMaxCols {
		cols = cardMaxCols
	}
	if cols > len(m.monitors) && len(m.monitors) > 0 {
		cols = len(m.monitors)
	}
	cardWidth = (m.width - cardGap*(cols-1)) / cols
	return cols, cardWidth
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
		case "l", "right":
			if m.selectedIndex < len(m.monitors)-1 {
				m.selectedIndex++
			}
			m.syncViewport()
		case "h", "left":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			m.syncViewport()
		case "j", "down":
			cols := m.numCols
			if cols < 1 {
				cols = 1
			}
			if m.selectedIndex+cols < len(m.monitors) {
				m.selectedIndex += cols
			} else {
				m.selectedIndex = len(m.monitors) - 1
			}
			m.syncViewport()
		case "k", "up":
			cols := m.numCols
			if cols < 1 {
				cols = 1
			}
			if m.selectedIndex-cols >= 0 {
				m.selectedIndex -= cols
			} else {
				m.selectedIndex = 0
			}
			m.syncViewport()
		case "g", "home":
			m.selectedIndex = 0
			m.syncViewport()
		case "G", "end":
			if len(m.monitors) > 0 {
				m.selectedIndex = len(m.monitors) - 1
			}
			m.syncViewport()
		case "r":
			m.loadData()
			m.syncViewport()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.syncViewport()

	case dashTickMsg:
		m.loadData()
		m.syncViewport()
		return m, dashTickCmd()
	}

	return m, nil
}

// syncViewport recomputes the viewport size and content and scrolls so the
// selected monitor card stays visible.
func (m *DashboardModel) syncViewport() {
	if !m.ready {
		return
	}

	if m.selectedIndex >= len(m.monitors) {
		m.selectedIndex = len(m.monitors) - 1
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}

	headerH := lipgloss.Height(m.headerView())
	helpH := lipgloss.Height(m.helpView())
	vpHeight := m.height - headerH - helpH
	if vpHeight < 1 {
		vpHeight = 1
	}

	m.viewport.Width = m.width
	m.viewport.Height = vpHeight

	cols, cardWidth := m.gridDims()
	m.numCols = cols

	cards := make([]string, len(m.monitors))
	for i, mon := range m.monitors {
		cards[i] = m.renderMonitorCard(mon, i == m.selectedIndex, cardWidth)
	}

	// Arrange cards into a grid of rows.
	var rows []string
	var rowHeights []int
	for i := 0; i < len(cards); i += cols {
		end := i + cols
		if end > len(cards) {
			end = len(cards)
		}
		row := joinCardsRow(cards[i:end], cardGap)
		rows = append(rows, row)
		rowHeights = append(rowHeights, lipgloss.Height(row))
	}
	m.viewport.SetContent(strings.Join(rows, "\n"))

	// Keep the selected card's row within the visible window.
	selRow := m.selectedIndex / cols
	top := 0
	for r := 0; r < selRow && r < len(rowHeights); r++ {
		top += rowHeights[r] + 1
	}
	selH := 0
	if selRow < len(rowHeights) {
		selH = rowHeights[selRow]
	}

	if top < m.viewport.YOffset {
		m.viewport.SetYOffset(top)
	} else if top+selH > m.viewport.YOffset+vpHeight {
		m.viewport.SetYOffset(top + selH - vpHeight)
	}
}

func joinCardsRow(cards []string, gapWidth int) string {
	if len(cards) == 0 {
		return ""
	}
	gap := strings.Repeat(" ", gapWidth)
	parts := make([]string, 0, len(cards)*2-1)
	for i, c := range cards {
		if i > 0 {
			parts = append(parts, gap)
		}
		parts = append(parts, c)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m DashboardModel) headerView() string {
	headerText := " 📊 STATPING DASHBOARD "
	header := dHeaderStyle.Render(headerText)
	statsText := dSubtitleStyle.Render(fmt.Sprintf("  %d monitors • Updated %s", len(m.monitors), m.lastUpdate.Format("15:04:05")))

	upCount, downCount, unknownCount := m.countStatus()
	summaryCards := m.renderSummaryCards(upCount, downCount, unknownCount)

	return header + statsText + "\n\n" + summaryCards
}

func (m DashboardModel) helpView() string {
	helpText := fmt.Sprintf("%s navigate • %s jump • %s refresh • %s quit",
		dHelpKeyStyle.Render("←↑↓→"),
		dHelpKeyStyle.Render("g/G"),
		dHelpKeyStyle.Render("r"),
		dHelpKeyStyle.Render("q"))
	return dHelpStyle.Render(helpText)
}

func (m DashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if len(m.monitors) == 0 {
		header := dHeaderStyle.Render(" 📊 STATPING DASHBOARD ")
		emptyMsg := lipgloss.NewStyle().
			Foreground(dColorGray).
			Italic(true).
			Render("  No monitors configured. Use 'statping add <url>' to add one.")
		return header + "\n\n" + emptyMsg
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		m.viewport.View(),
		m.helpView(),
	)
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

func (m DashboardModel) renderMonitorCard(mon storage.Monitor, selected bool, cardWidth int) string {
	results := m.checkResults[mon.ID]

	// Inner content width = total card width minus border (2) and padding (2*2).
	innerWidth := cardWidth - 6
	if innerWidth < 30 {
		innerWidth = 30
	}

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

	// Header row with status, name, and URL, truncated to fit the card.
	name := mon.Name
	maxName := innerWidth / 2
	if len(name) > maxName {
		name = name[:maxName-1] + "…"
	}
	urlBudget := innerWidth - 2 - len(name) - 2
	if urlBudget < 8 {
		urlBudget = 8
	}
	nameRow := fmt.Sprintf("%s %s  %s",
		statusStyle.Render(statusIcon),
		dMonitorNameStyle.Render(name),
		dUrlStyle.Render(truncateURL(mon.URL, urlBudget)))
	content.WriteString(nameRow)
	content.WriteString("\n\n")

	// Response time graph label
	content.WriteString(dMetricLabelStyle.Render("Response Time (last 60 checks):"))
	content.WriteString("\n")

	// Sparkline graph sized to the card (leaving room for the scale label).
	sparkWidth := innerWidth - 13
	if sparkWidth < 8 {
		sparkWidth = 8
	}
	graph := m.renderSparkline(results, sparkWidth)
	content.WriteString(graph)
	content.WriteString("\n\n")

	// Metrics row
	metricsRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderMetric("Uptime", fmt.Sprintf("%.1f%%", uptime), uptime >= 99),
		"  ",
		m.renderMetric("Avg", fmt.Sprintf("%dms", avgResponseTime), avgResponseTime < 500),
		"  ",
		m.renderMetric("Min", fmt.Sprintf("%dms", minResponseTime), true),
		"  ",
		m.renderMetric("Max", fmt.Sprintf("%dms", maxResponseTime), maxResponseTime < 1000),
		"  ",
		m.renderMetric("Checks", fmt.Sprintf("%d", len(results)), true),
	)
	content.WriteString(metricsRow)

	// Last check info (always rendered so cards in a row keep equal height)
	content.WriteString("\n\n")
	lastCheck := "Last check: never"
	if mon.LastCheckAt != nil {
		lastCheck = fmt.Sprintf("Last check: %s ago", formatTimeAgo(*mon.LastCheckAt))
	}
	content.WriteString(dMetricLabelStyle.Render(lastCheck))

	// Card styling based on status and selection
	var cardStyleFinal lipgloss.Style
	if selected {
		cardStyleFinal = dCardSelectedStyle.
			Width(innerWidth).
			Height(cardContentHeight).
			BorderForeground(dColorPurple)
	} else {
		borderColor := dColorDimGray
		if mon.CurrentStatus == "up" {
			borderColor = dColorGreen
		} else if mon.CurrentStatus == "down" {
			borderColor = dColorRed
		}
		cardStyleFinal = dCardStyle.
			Width(innerWidth).
			Height(cardContentHeight).
			BorderForeground(borderColor)
	}

	return cardStyleFinal.Render(content.String())
}

func (m DashboardModel) renderSparkline(results []storage.CheckResult, width int) string {
	if len(results) == 0 {
		return dMetricLabelStyle.Render("No data yet")
	}
	bars, maxTime := sparkline(results, width)
	scale := fmt.Sprintf(" (0-%dms)", maxTime)
	return bars + dMetricLabelStyle.Render(scale)
}

// sparkline renders response-time results (newest first) as colored blocks,
// oldest-to-newest left-to-right. Returns the rendered bars and the max time used for scaling.
func sparkline(results []storage.CheckResult, width int) (string, int64) {
	reversed := make([]storage.CheckResult, len(results))
	for i, r := range results {
		reversed[len(results)-1-i] = r
	}

	var maxTime int64 = 1
	for _, r := range reversed {
		if r.ResponseTime > maxTime {
			maxTime = r.ResponseTime
		}
	}

	var spark strings.Builder
	displayCount := width
	if len(reversed) < displayCount {
		displayCount = len(reversed)
	}

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

		normalized := float64(r.ResponseTime) / float64(maxTime)
		blockIdx := int(normalized * float64(len(dSparkBlocks)-1))
		if blockIdx >= len(dSparkBlocks) {
			blockIdx = len(dSparkBlocks) - 1
		}
		if blockIdx < 0 {
			blockIdx = 0
		}

		block := string(dSparkBlocks[blockIdx])
		if r.ResponseTime < 200 {
			spark.WriteString(dGraphGreenStyle.Render(block))
		} else if r.ResponseTime < 500 {
			spark.WriteString(dGraphYellowStyle.Render(block))
		} else {
			spark.WriteString(dGraphOrangeStyle.Render(block))
		}
	}

	return spark.String(), maxTime
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
