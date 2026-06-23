package components

import "testing"

func TestFormatNumberFull(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1000"},
		{4984, "4984"},
		{999999, "999999"},
		{1000000, "1000000"},
		{1500000, "1500000"},
	}
	for _, tt := range tests {
		got := FormatNumber(tt.input, true)
		if got != tt.want {
			t.Errorf("FormatNumber(%d, true) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatNumberAbbreviated(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{4984, "4.9k"},
		{999999, "999.9k"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{12500000, "12.5M"},
	}
	for _, tt := range tests {
		got := FormatNumber(tt.input, false)
		if got != tt.want {
			t.Errorf("FormatNumber(%d, false) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
