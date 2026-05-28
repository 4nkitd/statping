package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var terminalSizes = []struct{ w, h int }{
	{60, 15},
	{80, 24},
	{100, 30},
	{120, 40},
	{200, 50},
}

func TestListFitsTerminalWithManyMonitors(t *testing.T) {
	db := newTestDB(t)
	seedMonitors(t, db, 50)

	for _, sz := range terminalSizes {
		var m tea.Model = New(db)
		m, _ = m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})

		view := m.View()
		if got := lipgloss.Height(view); got > sz.h {
			t.Fatalf("[%dx%d] list rendered %d lines, exceeds terminal height", sz.w, sz.h, got)
		}
		if got := lipgloss.Width(view); got > sz.w {
			t.Fatalf("[%dx%d] list rendered width %d, exceeds terminal width", sz.w, sz.h, got)
		}
	}
}
