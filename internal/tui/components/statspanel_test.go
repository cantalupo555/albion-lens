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
