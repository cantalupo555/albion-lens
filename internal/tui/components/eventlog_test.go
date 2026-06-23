package components

import (
	"strings"
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/pkg/handlers"
)

func TestFormatEventMessageRespecWithSilver(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type: "respec",
		Data: &handlers.RespecEventData{
			Gained:       1000,
			PaidSilver:   500,
			SessionTotal: 3000,
		},
		Timestamp: time.Now(),
	}

	msg := e.formatEventMessage(event)

	if !strings.Contains(msg, "RESPEC") {
		t.Errorf("expected 'RESPEC' in message, got: %s", msg)
	}
	if !strings.Contains(msg, "+1000") {
		t.Errorf("expected '+1000' gained in message, got: %s", msg)
	}
	if !strings.Contains(msg, "500") {
		t.Errorf("expected silver cost '500' in message, got: %s", msg)
	}
}

func TestFormatEventMessageRespecWithoutSilver(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type: "respec",
		Data: &handlers.RespecEventData{
			Gained:       200,
			PaidSilver:   0,
			SessionTotal: 200,
		},
		Timestamp: time.Now(),
	}

	msg := e.formatEventMessage(event)

	if !strings.Contains(msg, "RESPEC") {
		t.Errorf("expected 'RESPEC' in message, got: %s", msg)
	}
	if !strings.Contains(msg, "+200") {
		t.Errorf("expected '+200' gained in message, got: %s", msg)
	}
	if strings.Contains(msg, "Silver cost") {
		t.Errorf("should not show 'Silver cost' when PaidSilver=0, got: %s", msg)
	}
}
