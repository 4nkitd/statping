package tui

import (
	"time"

	"github.com/ankityadav/statping/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
)

type sessionState int

const (
	listView sessionState = iota
	addView
	editView
	detailView
)

type Model struct {
	db     *storage.Database
	state  sessionState
	list   listModel
	form   formModel
	detail detailModel
	width  int
	height int
	err    error
}

type tickMsg time.Time

func New(db *storage.Database) Model {
	return Model{
		db:     db,
		state:  listView,
		list:   newListModel(db),
		form:   newFormModel(db),
		detail: newDetailModel(db),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.list.Init(),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == listView {
				return m, tea.Quit
			}
			m.state = listView
			m.list.loadMonitors()
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		if m.state == listView {
			m.list.loadMonitors()
		} else if m.state == detailView {
			m.detail.refresh()
		}
		return m, tickCmd()

	case MonitorSelectedMsg:
		m.state = detailView
		m.detail.setMonitor(msg.Monitor)
		return m, nil

	case AddMonitorMsg:
		m.state = addView
		m.form.reset()
		return m, nil

	case EditMonitorMsg:
		m.state = editView
		m.form.setMonitor(msg.Monitor)
		return m, nil

	case MonitorSavedMsg:
		m.state = listView
		m.list.loadMonitors()
		return m, nil

	case BackToListMsg:
		m.state = listView
		m.list.loadMonitors()
		return m, nil
	}

	switch m.state {
	case listView:
		listModel, listCmd := m.list.Update(msg)
		m.list = listModel
		cmds = append(cmds, listCmd)

	case addView, editView:
		formModel, formCmd := m.form.Update(msg)
		m.form = formModel
		cmds = append(cmds, formCmd)

	case detailView:
		detailModel, detailCmd := m.detail.Update(msg)
		m.detail = detailModel
		cmds = append(cmds, detailCmd)
	}

	return m, tea.Batch(append(cmds, cmd)...)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.state {
	case listView:
		return m.list.View()
	case addView, editView:
		return m.form.View()
	case detailView:
		return m.detail.View()
	default:
		return "Unknown state"
	}
}

type MonitorSelectedMsg struct {
	Monitor *storage.Monitor
}

type AddMonitorMsg struct{}

type EditMonitorMsg struct {
	Monitor *storage.Monitor
}

type MonitorSavedMsg struct{}

type BackToListMsg struct{}

func monitorSelected(m *storage.Monitor) tea.Cmd {
	return func() tea.Msg {
		return MonitorSelectedMsg{Monitor: m}
	}
}

func addMonitor() tea.Cmd {
	return func() tea.Msg {
		return AddMonitorMsg{}
	}
}

func editMonitor(m *storage.Monitor) tea.Cmd {
	return func() tea.Msg {
		return EditMonitorMsg{Monitor: m}
	}
}

func monitorSaved() tea.Cmd {
	return func() tea.Msg {
		return MonitorSavedMsg{}
	}
}

func backToList() tea.Cmd {
	return func() tea.Msg {
		return BackToListMsg{}
	}
}
