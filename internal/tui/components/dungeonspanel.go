package components

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/cantalupo555/albion-lens/pkg/handlers"
)

// DungeonsPanel displays the active dungeon run, a 90s close countdown, and a
// scrollable table of completed runs. It follows the immutable builder pattern.
type DungeonsPanel struct {
	viewport    viewport.Model
	runs        []*handlers.DungeonRun
	active      *handlers.DungeonRun
	runCount    int // last seen run count, for auto-scroll detection
	nowFunc     func() time.Time
	width       int
	height      int
	ready       bool
	fullNumbers bool
}

// NewDungeonsPanel creates a DungeonsPanel with sensible defaults.
func NewDungeonsPanel() DungeonsPanel {
	return DungeonsPanel{
		nowFunc:     time.Now,
		fullNumbers: false,
	}
}

// SetSize updates the dimensions of the dungeons panel.
func (d DungeonsPanel) SetSize(width, height int) DungeonsPanel {
	d.width = width
	d.height = height

	headerHeight := 2 // title + border
	footerHeight := 1 // border
	infoHeight := 5   // active run info block

	viewportHeight := height - headerHeight - footerHeight - infoHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	viewportWidth := width - 4 // borders + padding
	if viewportWidth < 10 {
		viewportWidth = 10
	}

	if !d.ready {
		d.viewport = viewport.New(viewport.WithWidth(viewportWidth), viewport.WithHeight(viewportHeight))
		d.ready = true
	} else {
		d.viewport.SetWidth(viewportWidth)
		d.viewport.SetHeight(viewportHeight)
	}

	return d
}

// SetFullNumbers sets whether to display full or abbreviated numbers.
func (d DungeonsPanel) SetFullNumbers(full bool) DungeonsPanel {
	d.fullNumbers = full
	return d
}

// SetNow sets the reference time used for the countdown display.
func (d DungeonsPanel) SetNow(t time.Time) DungeonsPanel {
	d.nowFunc = func() time.Time { return t }
	return d
}

// SetRuns updates the run data displayed by the panel. The viewport
// auto-scrolls to the bottom only when new runs have been appended since the
// last call, so manual scroll position is preserved during periodic ticks.
func (d DungeonsPanel) SetRuns(runs []*handlers.DungeonRun, active *handlers.DungeonRun) DungeonsPanel {
	newRuns := len(runs)
	d.runs = runs
	d.active = active

	if !d.ready {
		d.runCount = newRuns
		return d
	}
	d.viewport.SetContent(d.renderRunTable())
	if newRuns != d.runCount {
		d.viewport.GotoBottom()
	}
	d.runCount = newRuns
	return d
}

// ScrollUp scrolls the runs viewport up.
func (d DungeonsPanel) ScrollUp() DungeonsPanel {
	d.viewport.ScrollUp(1)
	return d
}

// ScrollDown scrolls the runs viewport down.
func (d DungeonsPanel) ScrollDown() DungeonsPanel {
	d.viewport.ScrollDown(1)
	return d
}

// Clear empties the run history.
func (d DungeonsPanel) Clear() DungeonsPanel {
	d.runs = nil
	d.active = nil
	if d.ready {
		d.viewport.SetContent(d.renderRunTable())
	}
	return d
}

// formatRunFields extracts and formats the common dungeon-run display fields
// (mode, faction, tier, duration). Shared by the run table and the active-run
// info block to keep formatting in a single place.
func (d DungeonsPanel) formatRunFields(r *handlers.DungeonRun, now time.Time) (mode, faction, tier, dur string) {
	mode = r.Mode.String()
	faction = r.Faction
	if faction == "" {
		faction = "—"
	}
	tier = "—"
	if r.Tier >= 0 {
		tier = fmt.Sprintf("T%d", r.Tier+1)
	}
	dur = formatElapsed(r.Duration(now))
	return
}

// renderRunTable builds the scrollable table of completed runs (newest first).
func (d DungeonsPanel) renderRunTable() string {
	if len(d.runs) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
		return emptyStyle.Render("No dungeon runs yet...")
	}

	now := d.nowFunc()
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Bold(true)

	doneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	closedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	header := headerStyle.Render(fmt.Sprintf(
		"%-7s %-8s %3s  %8s  %8s  %4s  %5s",
		"Mode", "Faction", "Tier", "Duration", "Fame", "Kills", "Loot",
	))

	lines := []string{header}

	for i := len(d.runs) - 1; i >= 0; i-- {
		r := d.runs[i]
		if r.Status == handlers.RunStatusActive {
			continue // active run shown in the info block, not the table
		}

		mode, faction, tier, dur := d.formatRunFields(r, now)
		fame := FormatNumber(r.Fame, d.fullNumbers)
		kills := fmt.Sprintf("%d", r.Kills)
		loot := fmt.Sprintf("%d", r.Loot)

		row := fmt.Sprintf("%-7s %-8s %3s  %8s  %8s  %4s  %5s",
			mode, faction, tier, dur, fame, kills, loot)

		if r.Status == handlers.RunStatusClosed {
			lines = append(lines, closedStyle.Render(row))
		} else {
			lines = append(lines, doneStyle.Render(row))
		}
	}

	return strings.Join(lines, "\n")
}

// View renders the dungeons panel.
func (d DungeonsPanel) View() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(d.width - 2).
		Height(d.height - 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("119"))

	countdownStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	now := d.nowFunc()

	// Active run info block
	var activeLines []string
	if d.active != nil {
		r := d.active
		mode, faction, tier, dur := d.formatRunFields(r, now)

		activeLines = append(activeLines, fmt.Sprintf("%s  %s (%s, %s, %s)",
			labelStyle.Render("Active:"),
			activeStyle.Render(mode),
			faction, tier, dur,
		))

		// Countdown
		remaining := r.CloseAt.Sub(now)
		if remaining < 0 {
			remaining = 0
		}
		activeLines = append(activeLines, fmt.Sprintf("%s  %s",
			labelStyle.Render("Close in:"),
			countdownStyle.Render(formatElapsed(remaining)),
		))
	} else {
		emptyActive := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
		activeLines = append(activeLines, emptyActive.Render("No active dungeon run."))
		activeLines = append(activeLines, "")
	}

	// Run history title
	historyTitle := labelStyle.Render("Run History:")

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Dungeons"),
		activeLines[0],
		activeLines[1],
		"",
		historyTitle,
		d.viewport.View(),
	)

	return boxStyle.Render(content)
}
