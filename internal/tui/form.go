package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ankityadav/statping/internal/config"
	"github.com/ankityadav/statping/internal/storage"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type formModel struct {
	db         *storage.Database
	monitor    *storage.Monitor
	inputs     []textinput.Model
	focusIndex int
	isEdit     bool
	err        error
}

const (
	inputName = iota
	inputURL
	inputInterval
	inputTimeout
	inputExpectedCodes
	inputKeywords
)

func newFormModel(db *storage.Database) formModel {
	inputs := make([]textinput.Model, 6)

	inputs[inputName] = textinput.New()
	inputs[inputName].Placeholder = "My Website"
	inputs[inputName].Focus()
	inputs[inputName].CharLimit = 100
	inputs[inputName].Width = 50

	inputs[inputURL] = textinput.New()
	inputs[inputURL].Placeholder = "https://example.com"
	inputs[inputURL].CharLimit = 500
	inputs[inputURL].Width = 50

	inputs[inputInterval] = textinput.New()
	inputs[inputInterval].Placeholder = "60"
	inputs[inputInterval].CharLimit = 5
	inputs[inputInterval].Width = 20

	inputs[inputTimeout] = textinput.New()
	inputs[inputTimeout].Placeholder = "10"
	inputs[inputTimeout].CharLimit = 3
	inputs[inputTimeout].Width = 20

	inputs[inputExpectedCodes] = textinput.New()
	inputs[inputExpectedCodes].Placeholder = "200,201,204"
	inputs[inputExpectedCodes].CharLimit = 50
	inputs[inputExpectedCodes].Width = 50

	inputs[inputKeywords] = textinput.New()
	inputs[inputKeywords].Placeholder = "Success,OK (comma-separated, optional)"
	inputs[inputKeywords].CharLimit = 200
	inputs[inputKeywords].Width = 50

	return formModel{
		db:     db,
		inputs: inputs,
	}
}

func (m *formModel) reset() {
	m.monitor = nil
	m.isEdit = false
	m.focusIndex = 0
	m.err = nil

	m.inputs[inputName].SetValue("")
	m.inputs[inputURL].SetValue("")
	m.inputs[inputInterval].SetValue(fmt.Sprintf("%d", config.DefaultCheckInterval))
	m.inputs[inputTimeout].SetValue(fmt.Sprintf("%d", config.DefaultTimeout))
	m.inputs[inputExpectedCodes].SetValue("200")
	m.inputs[inputKeywords].SetValue("")

	m.inputs[inputName].Focus()
	for i := 1; i < len(m.inputs); i++ {
		m.inputs[i].Blur()
	}
}

func (m *formModel) setMonitor(monitor *storage.Monitor) {
	m.monitor = monitor
	m.isEdit = true
	m.focusIndex = 0
	m.err = nil

	m.inputs[inputName].SetValue(monitor.Name)
	m.inputs[inputURL].SetValue(monitor.URL)
	m.inputs[inputInterval].SetValue(fmt.Sprintf("%d", monitor.CheckInterval))
	m.inputs[inputTimeout].SetValue(fmt.Sprintf("%d", monitor.Timeout))
	m.inputs[inputExpectedCodes].SetValue(monitor.ExpectedCodes)
	m.inputs[inputKeywords].SetValue(monitor.Keywords)

	m.inputs[inputName].Focus()
	for i := 1; i < len(m.inputs); i++ {
		m.inputs[i].Blur()
	}
}

func (m formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, backToList()

		case "tab", "down", "j":
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}
			return m, m.updateFocus()

		case "shift+tab", "up", "k":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			return m, m.updateFocus()

		case "enter":
			if m.focusIndex == len(m.inputs)-1 {
				return m, m.save()
			}
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}
			return m, m.updateFocus()
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *formModel) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	for i := 0; i < len(m.inputs); i++ {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}

	return tea.Batch(cmds...)
}

func (m *formModel) save() tea.Cmd {
	name := strings.TrimSpace(m.inputs[inputName].Value())
	url := strings.TrimSpace(m.inputs[inputURL].Value())

	if name == "" {
		m.err = fmt.Errorf("name is required")
		return nil
	}

	if url == "" {
		m.err = fmt.Errorf("URL is required")
		return nil
	}

	interval, err := strconv.Atoi(m.inputs[inputInterval].Value())
	if err != nil || interval < 1 {
		interval = config.DefaultCheckInterval
	}

	timeout, err := strconv.Atoi(m.inputs[inputTimeout].Value())
	if err != nil || timeout < 1 {
		timeout = config.DefaultTimeout
	}

	expectedCodes := strings.TrimSpace(m.inputs[inputExpectedCodes].Value())
	if expectedCodes == "" {
		expectedCodes = "200"
	}

	keywords := strings.TrimSpace(m.inputs[inputKeywords].Value())

	if m.isEdit && m.monitor != nil {
		m.monitor.Name = name
		m.monitor.URL = url
		m.monitor.CheckInterval = interval
		m.monitor.Timeout = timeout
		m.monitor.ExpectedCodes = expectedCodes
		m.monitor.Keywords = keywords

		if err := m.db.UpdateMonitor(m.monitor); err != nil {
			m.err = err
			return nil
		}
	} else {
		monitor := &storage.Monitor{
			Name:          name,
			URL:           url,
			CheckInterval: interval,
			Timeout:       timeout,
			ExpectedCodes: expectedCodes,
			Keywords:      keywords,
			Enabled:       true,
		}

		if err := m.db.CreateMonitor(monitor); err != nil {
			m.err = err
			return nil
		}
	}

	return monitorSaved()
}

func (m formModel) View() string {
	var b strings.Builder

	title := "Add Monitor"
	if m.isEdit {
		title = "Edit Monitor"
	}

	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	labels := []string{
		"Name:",
		"URL:",
		"Check Interval (seconds):",
		"Timeout (seconds):",
		"Expected Status Codes:",
		"Keywords (comma-separated):",
	}

	for i, input := range m.inputs {
		b.WriteString(lipgloss.NewStyle().Bold(true).Render(labels[i]))
		b.WriteString("\n")
		b.WriteString(input.View())
		b.WriteString("\n\n")
	}

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		"tab/j: next • shift+tab/k: previous • enter: save • esc: cancel",
	)
	b.WriteString(help)

	return baseStyle.Render(b.String())
}
