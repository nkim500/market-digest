package viewport

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func rows(n int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = "row-" + string(rune('A'+i%26))
	}
	return out
}

func TestViewportCursorAlwaysVisible(t *testing.T) {
	m := New().SetHeight(5).SetRows(rows(20))
	// Move cursor down 15 times.
	for i := 0; i < 15; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.Cursor() != 15 {
		t.Fatalf("cursor should be 15, got %d", m.Cursor())
	}
	// The visible window should include row 15.
	view := m.View()
	if !strings.Contains(view, rows(20)[15]) {
		t.Errorf("visible window missing cursor row; view=%q", view)
	}
}

func TestViewportPgDnAdvancesHalfPage(t *testing.T) {
	m := New().SetHeight(10).SetRows(rows(50))
	startCursor := m.Cursor()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	got := m.Cursor() - startCursor
	if got < 4 || got > 6 {
		t.Errorf("PgDn should advance ~5 (half of height=10); advanced %d", got)
	}
}

func TestViewportEndJumpsToLastRow(t *testing.T) {
	m := New().SetHeight(5).SetRows(rows(100))
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.Cursor() != 99 {
		t.Errorf("End should move cursor to 99, got %d", m.Cursor())
	}
}

func TestViewportHomeJumpsToFirstRow(t *testing.T) {
	m := New().SetHeight(5).SetRows(rows(100))
	for i := 0; i < 50; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.Cursor() != 0 {
		t.Errorf("Home should reset cursor to 0, got %d", m.Cursor())
	}
}

func TestViewportEmptyRows(t *testing.T) {
	m := New().SetHeight(5).SetRows(nil)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor() != 0 {
		t.Errorf("empty rows should keep cursor at 0, got %d", m.Cursor())
	}
	view := m.View()
	if view != "" {
		t.Errorf("empty rows view should be empty, got %q", view)
	}
}
