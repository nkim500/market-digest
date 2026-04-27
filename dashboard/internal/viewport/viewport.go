// Package viewport is a reusable scroll/cursor wrapper for any list-of-rows
// screen in the dashboard. It handles arrow keys, PgUp/PgDn, Home/End, and
// keeps the cursor visible as the window scrolls.
package viewport

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	rows   []string
	cursor int
	top    int
	height int
	// usableHeight is the row count actually shown (subtract 2 for header/footer by default).
	reservedLines int
}

func New() Model {
	return Model{reservedLines: 2}
}

func (m Model) SetRows(rs []string) Model {
	m.rows = rs
	if m.cursor >= len(rs) {
		m.cursor = 0
	}
	return m
}

func (m Model) SetHeight(h int) Model {
	m.height = h
	return m
}

func (m Model) Cursor() int { return m.cursor }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if len(m.rows) == 0 {
		return m, nil
	}
	switch km.Type {
	case tea.KeyUp:
		m = m.move(-1)
	case tea.KeyDown:
		m = m.move(1)
	case tea.KeyPgUp:
		m = m.move(-m.pageStep())
	case tea.KeyPgDown:
		m = m.move(m.pageStep())
	case tea.KeyHome:
		m = m.moveTo(0)
	case tea.KeyEnd:
		m = m.moveTo(len(m.rows) - 1)
	case tea.KeyRunes:
		switch string(km.Runes) {
		case "k":
			m = m.move(-1)
		case "j":
			m = m.move(1)
		case "g":
			m = m.moveTo(0)
		case "G":
			m = m.moveTo(len(m.rows) - 1)
		}
	case tea.KeyCtrlU:
		m = m.move(-m.pageStep())
	case tea.KeyCtrlD:
		m = m.move(m.pageStep())
	}
	return m, nil
}

func (m Model) View() string {
	if len(m.rows) == 0 {
		return ""
	}
	usable := m.usableHeight()
	end := m.top + usable
	if end > len(m.rows) {
		end = len(m.rows)
	}
	slice := m.rows[m.top:end]
	return strings.Join(slice, "\n")
}

func (m Model) move(delta int) Model {
	return m.moveTo(m.cursor + delta)
}

func (m Model) moveTo(idx int) Model {
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.rows) {
		idx = len(m.rows) - 1
	}
	m.cursor = idx
	usable := m.usableHeight()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+usable {
		m.top = m.cursor - usable + 1
	}
	if m.top < 0 {
		m.top = 0
	}
	return m
}

func (m Model) pageStep() int {
	step := m.usableHeight() / 2
	if step < 1 {
		step = 1
	}
	return step
}

func (m Model) usableHeight() int {
	u := m.height - m.reservedLines
	if u < 1 {
		u = 10
	}
	return u
}
