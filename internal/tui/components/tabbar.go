package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// tabActiveStyle and tabInactiveStyle mirror the shared styles in the tui
// package. They are redefined here because the components package cannot import
// the tui package (circular dependency); the values are kept identical.
var (
	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Padding(0, 1)
)

// TabBar renders a horizontal row of tabs and tracks the active one. It follows
// the immutable builder pattern: all mutator methods return a new TabBar.
type TabBar struct {
	tabs   []string
	active int
	width  int
}

// NewTabBar creates a TabBar with the default tabs (Dashboard, Zone).
func NewTabBar() TabBar {
	return TabBar{
		tabs: []string{"Dashboard", "Zone"},
	}
}

// Active returns the index of the currently active tab.
func (t TabBar) Active() int {
	return t.active
}

// SetActive returns a TabBar with the active tab set to i, clamped to valid
// bounds.
func (t TabBar) SetActive(i int) TabBar {
	t.active = i
	if t.active < 0 {
		t.active = 0
	}
	if t.active >= len(t.tabs) {
		t.active = len(t.tabs) - 1
	}
	return t
}

// Next advances the active tab forward, wrapping around to the first tab.
func (t TabBar) Next() TabBar {
	t.active = (t.active + 1) % len(t.tabs)
	return t
}

// Prev moves the active tab backward, wrapping around to the last tab.
func (t TabBar) Prev() TabBar {
	t.active--
	if t.active < 0 {
		t.active = len(t.tabs) - 1
	}
	return t
}

// SetWidth sets the total render width for the tab bar.
func (t TabBar) SetWidth(w int) TabBar {
	t.width = w
	return t
}

// View renders the tab bar as a horizontal row. The active tab is bold with an
// inverted (filled) background; inactive tabs are dimmed.
func (t TabBar) View() string {
	rendered := make([]string, len(t.tabs))
	for i, label := range t.tabs {
		if i == t.active {
			rendered[i] = tabActiveStyle.Render(" " + label + " ")
		} else {
			rendered[i] = tabInactiveStyle.Render(" " + label + " ")
		}
	}

	return strings.Join(rendered, "")
}
