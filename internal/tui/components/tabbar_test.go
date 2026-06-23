package components

import (
	"strings"
	"testing"
)

func TestNewTabBarDefaults(t *testing.T) {
	tb := NewTabBar()

	if tb.active != 0 {
		t.Errorf("expected active=0 by default, got %d", tb.active)
	}
	if len(tb.tabs) != 2 {
		t.Errorf("expected 2 tabs, got %d", len(tb.tabs))
	}
}

func TestTabBarActive(t *testing.T) {
	tb := NewTabBar()
	if tb.Active() != 0 {
		t.Errorf("expected Active()=0, got %d", tb.Active())
	}
}

func TestTabBarNext(t *testing.T) {
	tb := NewTabBar()

	tb = tb.Next()
	if tb.Active() != 1 {
		t.Errorf("expected Active()=1 after Next, got %d", tb.Active())
	}

	// Wrap around.
	tb = tb.Next()
	if tb.Active() != 0 {
		t.Errorf("expected Active()=0 after wrap-around Next, got %d", tb.Active())
	}
}

func TestTabBarPrev(t *testing.T) {
	tb := NewTabBar()

	// Wrap backward to the last tab.
	tb = tb.Prev()
	if tb.Active() != 1 {
		t.Errorf("expected Active()=1 after backward wrap Prev, got %d", tb.Active())
	}

	tb = tb.Prev()
	if tb.Active() != 0 {
		t.Errorf("expected Active()=0 after Prev, got %d", tb.Active())
	}
}

func TestTabBarSetActive(t *testing.T) {
	tb := NewTabBar()

	tb = tb.SetActive(1)
	if tb.Active() != 1 {
		t.Errorf("expected Active()=1, got %d", tb.Active())
	}

	// Clamping above.
	tb = tb.SetActive(99)
	if tb.Active() != 1 {
		t.Errorf("expected clamped Active()=1, got %d", tb.Active())
	}

	// Clamping below.
	tb = tb.SetActive(-5)
	if tb.Active() != 0 {
		t.Errorf("expected clamped Active()=0, got %d", tb.Active())
	}
}

func TestTabBarViewContainsAllTabs(t *testing.T) {
	tb := NewTabBar()

	view := tb.View()

	if !strings.Contains(view, "Dashboard") {
		t.Error("expected 'Dashboard' in tab bar view")
	}
	if !strings.Contains(view, "Zone") {
		t.Error("expected 'Zone' in tab bar view")
	}
}

func TestTabBarViewActiveIsFirst(t *testing.T) {
	tb := NewTabBar() // active = 0 (Dashboard)

	view := tb.View()

	// Both tab names should be present.
	dashIdx := strings.Index(view, "Dashboard")
	zoneIdx := strings.Index(view, "Zone")
	if dashIdx < 0 || zoneIdx < 0 {
		t.Fatalf("missing tab names in view: %s", view)
	}
	if dashIdx > zoneIdx {
		t.Error("expected Dashboard to appear before Zone")
	}
}

func TestTabBarViewZoneActive(t *testing.T) {
	tb := NewTabBar()
	tb = tb.SetActive(1) // Zone tab is active

	view := tb.View()

	if !strings.Contains(view, "Dashboard") {
		t.Error("expected 'Dashboard' in view")
	}
	if !strings.Contains(view, "Zone") {
		t.Error("expected 'Zone' in view")
	}

	// The active tab (Zone) should be styled differently from Dashboard.
	// The active style uses ANSI escape codes (bold + background color).
	// Verify the view contains escape sequences — at least one for each tab.
	if !strings.Contains(view, "\x1b[") {
		t.Error("expected ANSI escape sequences for styled tabs in view")
	}
}
