package components

import (
	"strings"
	"testing"
)

func TestStatsPanelSetRespec(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetRespec(5000)

	if s.respec != 5000 {
		t.Errorf("expected respec 5000, got %d", s.respec)
	}
}

func TestStatsPanelSetRespecSilver(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetRespecSilver(2500)

	if s.respecSilver != 2500 {
		t.Errorf("expected respecSilver 2500, got %d", s.respecSilver)
	}
}

func TestStatsPanelResetClearsRespec(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetRespec(5000)
	s = s.SetRespecSilver(2500)
	s = s.Reset()

	if s.respec != 0 {
		t.Errorf("expected respec 0 after reset, got %d", s.respec)
	}
	if s.respecSilver != 0 {
		t.Errorf("expected respecSilver 0 after reset, got %d", s.respecSilver)
	}
}

func TestStatsPanelViewShowsRespec(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetFullNumbers(true)
	s = s.SetRespec(7500)
	s = s.SetSize(30, 12)

	output := s.View()

	if !strings.Contains(output, "Respec") {
		t.Error("View output should contain 'Respec' label")
	}
	if !strings.Contains(output, "7500") {
		t.Error("View output should contain respec value 7500")
	}
	if strings.Contains(output, "silver") {
		t.Error("View should not show silver when respecSilver is 0")
	}
}

func TestStatsPanelViewShowsRespecSilverWhenNonZero(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetFullNumbers(true)
	s = s.SetRespec(7500)
	s = s.SetRespecSilver(300)
	s = s.SetSize(35, 12)

	output := s.View()

	if !strings.Contains(output, "silver") {
		t.Error("View should show silver when respecSilver > 0")
	}
	if !strings.Contains(output, "300") {
		t.Error("View should contain silver value 300")
	}
}

// ============================================
// Fame tests
// ============================================

func TestStatsPanelSetFame(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetFame(5000)

	if s.fame != 5000 {
		t.Errorf("expected fame 5000, got %d", s.fame)
	}
}

func TestStatsPanelAddFame(t *testing.T) {
	s := NewStatsPanel()
	s = s.AddFame(100)
	s = s.AddFame(200)

	if s.fame != 300 {
		t.Errorf("expected accumulated fame 300, got %d", s.fame)
	}
}

// ============================================
// Silver tests
// ============================================

func TestStatsPanelSetSilver(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetSilver(8000)

	if s.silver != 8000 {
		t.Errorf("expected silver 8000, got %d", s.silver)
	}
}

func TestStatsPanelAddSilver(t *testing.T) {
	s := NewStatsPanel()
	s = s.AddSilver(500)
	s = s.AddSilver(250)

	if s.silver != 750 {
		t.Errorf("expected accumulated silver 750, got %d", s.silver)
	}
}

// ============================================
// Kill/Death/Loot counter tests
// ============================================

func TestStatsPanelIncrKills(t *testing.T) {
	s := NewStatsPanel()
	s = s.IncrKills()
	s = s.IncrKills()
	s = s.IncrKills()

	if s.kills != 3 {
		t.Errorf("expected kills 3, got %d", s.kills)
	}
}

func TestStatsPanelIncrDeaths(t *testing.T) {
	s := NewStatsPanel()
	s = s.IncrDeaths()
	s = s.IncrDeaths()

	if s.deaths != 2 {
		t.Errorf("expected deaths 2, got %d", s.deaths)
	}
}

func TestStatsPanelIncrLoot(t *testing.T) {
	s := NewStatsPanel()
	s = s.IncrLoot()
	s = s.IncrLoot()
	s = s.IncrLoot()
	s = s.IncrLoot()

	if s.lootCount != 4 {
		t.Errorf("expected lootCount 4, got %d", s.lootCount)
	}
}

func TestStatsPanelSetKills(t *testing.T) {
	s := NewStatsPanel()
	s = s.IncrKills()
	s = s.IncrKills()
	s = s.SetKills(10)
	if s.kills != 10 {
		t.Errorf("SetKills: expected 10 (absolute), got %d", s.kills)
	}
	// Incr after Set adds on top of the absolute value.
	s = s.IncrKills()
	if s.kills != 11 {
		t.Errorf("IncrKills after SetKills: expected 11, got %d", s.kills)
	}
}

func TestStatsPanelSetDeaths(t *testing.T) {
	s := NewStatsPanel()
	s = s.IncrDeaths()
	s = s.SetDeaths(5)
	if s.deaths != 5 {
		t.Errorf("SetDeaths: expected 5 (absolute), got %d", s.deaths)
	}
	s = s.IncrDeaths()
	if s.deaths != 6 {
		t.Errorf("IncrDeaths after SetDeaths: expected 6, got %d", s.deaths)
	}
}

func TestStatsPanelSetLoot(t *testing.T) {
	s := NewStatsPanel()
	s = s.IncrLoot()
	s = s.IncrLoot()
	s = s.SetLoot(8)
	if s.lootCount != 8 {
		t.Errorf("SetLoot: expected 8 (absolute), got %d", s.lootCount)
	}
	s = s.IncrLoot()
	if s.lootCount != 9 {
		t.Errorf("IncrLoot after SetLoot: expected 9, got %d", s.lootCount)
	}
}

// ============================================
// Reset tests (full)
// ============================================

func TestStatsPanelResetClearsAll(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetFame(5000)
	s = s.SetSilver(3000)
	s = s.SetRespec(1000)
	s = s.SetRespecSilver(500)
	s = s.IncrKills()
	s = s.IncrDeaths()
	s = s.IncrLoot()

	s = s.Reset()

	if s.fame != 0 {
		t.Errorf("expected fame 0 after reset, got %d", s.fame)
	}
	if s.silver != 0 {
		t.Errorf("expected silver 0 after reset, got %d", s.silver)
	}
	if s.respec != 0 {
		t.Errorf("expected respec 0 after reset, got %d", s.respec)
	}
	if s.respecSilver != 0 {
		t.Errorf("expected respecSilver 0 after reset, got %d", s.respecSilver)
	}
	if s.kills != 0 {
		t.Errorf("expected kills 0 after reset, got %d", s.kills)
	}
	if s.deaths != 0 {
		t.Errorf("expected deaths 0 after reset, got %d", s.deaths)
	}
	if s.lootCount != 0 {
		t.Errorf("expected lootCount 0 after reset, got %d", s.lootCount)
	}
}

// ============================================
// View tests (all stats)
// ============================================

func TestStatsPanelViewShowsAllLabels(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetFullNumbers(true)
	s = s.SetFame(1000)
	s = s.SetSilver(2000)
	s = s.IncrKills()
	s = s.IncrDeaths()
	s = s.IncrLoot()
	s = s.SetSize(40, 14)

	output := s.View()

	labels := []string{"Fame", "Silver", "Respec", "Kills", "Deaths", "Loot"}
	for _, label := range labels {
		if !strings.Contains(output, label) {
			t.Errorf("expected '%s' label in view", label)
		}
	}
}

func TestStatsPanelViewAbbreviatedNumbers(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetFullNumbers(false)
	s = s.SetFame(1500000)
	s = s.SetSize(40, 14)

	output := s.View()

	if !strings.Contains(output, "1.5M") {
		t.Error("expected abbreviated '1.5M' in view with fullNumbers=false")
	}
}

func TestStatsPanelViewFullNumbers(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetFullNumbers(true)
	s = s.SetFame(1500000)
	s = s.SetSize(40, 14)

	output := s.View()

	if !strings.Contains(output, "1500000") {
		t.Error("expected full '1500000' in view with fullNumbers=true")
	}
}

func TestStatsPanelViewNegativeSilverAbbreviated(t *testing.T) {
	s := NewStatsPanel()
	s = s.SetFullNumbers(false)
	s = s.SetRespec(7500)
	s = s.SetRespecSilver(2500)
	s = s.SetSize(35, 12)

	output := s.View()

	if !strings.Contains(output, "-2.5k") {
		t.Errorf("View should show negative abbreviated silver (-2.5k), got: %s", output)
	}
}

func TestFormatAbbreviatedPreservesSign(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{2500, "2.5k"},
		{-2500, "2.5k"},
		{2500000, "2.5M"},
		{-2500000, "2.5M"},
		{500, "500"},
		{-500, "500"},
	}
	for _, c := range cases {
		got := formatAbbreviated(c.input)
		if got != c.want {
			t.Errorf("formatAbbreviated(%d) = %q, want %q", c.input, got, c.want)
		}
	}
}
