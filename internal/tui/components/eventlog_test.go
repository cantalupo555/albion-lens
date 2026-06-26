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

// ============================================
// formatEventMessage: other event types
// ============================================

func TestFormatEventMessageFame(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type: "fame",
		Data: &handlers.FameEventData{
			Gained:  500,
			Total:   5000,
			Session: 2000,
		},
		Timestamp: time.Now(),
	}

	msg := e.formatEventMessage(event)

	if !strings.Contains(msg, "FAME") {
		t.Errorf("expected 'FAME' in message, got: %s", msg)
	}
	if !strings.Contains(msg, "+500") {
		t.Errorf("expected '+500' gained in message, got: %s", msg)
	}
	if !strings.Contains(msg, "5000") {
		t.Errorf("expected '5000' total in message, got: %s", msg)
	}
}

func TestFormatEventMessageSilver(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type: "silver",
		Data: &handlers.SilverEventData{
			Amount:     750,
			Session:    3000,
			LootedBy:   "PlayerA",
			LootedFrom: "BossNPC",
		},
		Timestamp: time.Now(),
	}

	msg := e.formatEventMessage(event)

	if !strings.Contains(msg, "PlayerA") {
		t.Errorf("expected 'PlayerA' (looter) in message, got: %s", msg)
	}
	if !strings.Contains(msg, "750") {
		t.Errorf("expected '750' amount in message, got: %s", msg)
	}
	if !strings.Contains(msg, "BossNPC") {
		t.Errorf("expected 'BossNPC' (source) in message, got: %s", msg)
	}
}

func TestFormatEventMessageLoot(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type: "loot",
		Data: &handlers.LootEventData{
			LootedBy:   "PlayerX",
			ItemName:   "Abyssal Sword",
			Quantity:   3,
			LootedFrom: "Dungeon Boss",
		},
		Timestamp: time.Now(),
	}

	msg := e.formatEventMessage(event)

	if !strings.Contains(msg, "PlayerX") {
		t.Errorf("expected looter name in message, got: %s", msg)
	}
	if !strings.Contains(msg, "Abyssal Sword") {
		t.Errorf("expected item name in message, got: %s", msg)
	}
	if !strings.Contains(msg, "x3") {
		t.Errorf("expected 'x3' quantity in message, got: %s", msg)
	}
}

func TestFormatEventMessageKill(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type: "kill",
		Data: &handlers.KillEventData{
			SessionKills: 5,
		},
		Timestamp: time.Now(),
	}

	msg := e.formatEventMessage(event)

	if !strings.Contains(msg, "Killed") {
		t.Errorf("expected 'Killed' in message, got: %s", msg)
	}
	if !strings.Contains(msg, "5") {
		t.Errorf("expected session kills count '5' in message, got: %s", msg)
	}
}

func TestFormatEventMessageDeathWithKiller(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type: "death",
		Data: &handlers.DeathEventData{
			Victim: "NoobSlayer",
			Killer: "BossEnemy",
		},
		Timestamp: time.Now(),
	}

	msg := e.formatEventMessage(event)

	if !strings.Contains(msg, "NoobSlayer") {
		t.Errorf("expected victim name in message, got: %s", msg)
	}
	if !strings.Contains(msg, "BossEnemy") {
		t.Errorf("expected killer name in message, got: %s", msg)
	}
}

func TestFormatEventMessageDeathWithoutKiller(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type: "death",
		Data: &handlers.DeathEventData{
			Victim: "UnluckyPlayer",
			Killer: "",
		},
		Timestamp: time.Now(),
	}

	msg := e.formatEventMessage(event)

	if !strings.Contains(msg, "UnluckyPlayer") {
		t.Errorf("expected victim name in message, got: %s", msg)
	}
	if strings.Contains(msg, "Killed by") {
		t.Errorf("should not show 'Killed by' when Killer is empty, got: %s", msg)
	}
}

func TestFormatEventMessageFallback(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type:    "unknown_type",
		Message: "Custom fallback message",
	}

	msg := e.formatEventMessage(event)

	if msg != "Custom fallback message" {
		t.Errorf("expected fallback message, got: %s", msg)
	}
}

func TestFormatEventMessageZoneWithPrevious(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type: "zone",
		Data: &handlers.ZoneEventData{
			MapType:  handlers.MapTypeIsland,
			Display:  "Island — Farm",
			Previous: handlers.MapTypeRandomDungeon,
		},
		Timestamp: time.Now(),
	}

	msg := e.formatEventMessage(event)

	if !strings.Contains(msg, "Island — Farm") {
		t.Errorf("expected zone label in message, got: %s", msg)
	}
	if !strings.Contains(msg, "was:") {
		t.Errorf("expected 'was:' suffix with previous zone, got: %s", msg)
	}
	if !strings.Contains(msg, "Random Dungeon") {
		t.Errorf("expected previous zone name in message, got: %s", msg)
	}
}

func TestFormatEventMessageZoneFirstTransition(t *testing.T) {
	e := EventLog{fullNumbers: true}

	event := Event{
		Type: "zone",
		Data: &handlers.ZoneEventData{
			MapType:  handlers.MapTypeIsland,
			Display:  "Island — Farm",
			Previous: handlers.MapTypeUnknown,
		},
		Timestamp: time.Now(),
	}

	msg := e.formatEventMessage(event)

	if !strings.Contains(msg, "Island — Farm") {
		t.Errorf("expected zone label in message, got: %s", msg)
	}
	if strings.Contains(msg, "was:") {
		t.Errorf("should not show 'was:' on first transition, got: %s", msg)
	}
}

// ============================================
// AddEvents / Clear / SetFullNumbers tests
// ============================================

func TestAddEventsBatch(t *testing.T) {
	e := NewEventLog()
	e = e.SetSize(60, 20)

	events := []Event{
		{Type: "fame", Message: "fame event", Timestamp: time.Now()},
		{Type: "silver", Message: "silver event", Timestamp: time.Now()},
		{Type: "loot", Message: "loot event", Timestamp: time.Now()},
	}

	e = e.AddEvents(events)

	view := e.View()
	if !strings.Contains(view, "fame event") {
		t.Error("expected 'fame event' in view after AddEvents")
	}
	if !strings.Contains(view, "silver event") {
		t.Error("expected 'silver event' in view after AddEvents")
	}
}

func TestAddEventsEmpty(t *testing.T) {
	e := NewEventLog()
	e = e.SetSize(60, 20)

	// Adding empty slice should not panic
	e = e.AddEvents([]Event{})
}

func TestAddEventsTrimsAtMaxEvents(t *testing.T) {
	e := NewEventLog()
	e = e.SetSize(120, 30)

	// Add maxEvents+1 events
	events := make([]Event, maxEvents+1)
	for i := range events {
		events[i] = Event{
			Type:    "info",
			Message: "filler",
		}
	}
	events[maxEvents].Message = "newest event"

	e = e.AddEvents(events)

	// Should only keep the newest maxEvents events
	if len(e.events) != maxEvents {
		t.Errorf("expected %d events after trim, got %d", maxEvents, len(e.events))
	}

	// The newest event should be present
	view := e.View()
	if !strings.Contains(view, "newest event") {
		t.Error("expected newest event to be retained after trim")
	}
}

func TestClear(t *testing.T) {
	e := NewEventLog()
	e = e.SetSize(60, 20)

	e = e.AddEvents([]Event{
		{Type: "info", Message: "test event", Timestamp: time.Now()},
	})

	e = e.Clear()

	if len(e.events) != 0 {
		t.Errorf("expected 0 events after clear, got %d", len(e.events))
	}
	if len(e.renderedLines) != 0 {
		t.Errorf("expected 0 rendered lines after clear, got %d", len(e.renderedLines))
	}
}

func TestSetFullNumbersReRender(t *testing.T) {
	e := NewEventLog()
	e = e.SetSize(80, 20)

	// Add event with full numbers (default)
	e = e.AddEvents([]Event{
		{
			Type:      "fame",
			Data:      &handlers.FameEventData{Gained: 1500, Total: 5000, Session: 3000},
			Timestamp: time.Now(),
		},
	})

	viewFull := e.View()
	if !strings.Contains(viewFull, "1500") {
		t.Error("expected full number '1500' with fullNumbers=true")
	}

	// Toggle to abbreviated
	e = e.SetFullNumbers(false)

	viewAbb := e.View()
	if !strings.Contains(viewAbb, "1.5k") {
		t.Error("expected abbreviated '1.5k' with fullNumbers=false")
	}
}

// TestAddEventsWarningRendered verifies that warning events are rendered in
// the log with their message text and the amber+bold style defined for the
// "warning" case in renderSingleEvent (Foreground 214 + Bold).
func TestAddEventsWarningRendered(t *testing.T) {
	e := NewEventLog()
	e = e.SetSize(80, 20)

	e = e.AddEvents([]Event{
		{
			Type:      "warning",
			Message:   "Could not capture on eth0: permission denied",
			Timestamp: time.Now(),
		},
	})

	view := e.View()
	if !strings.Contains(view, "Could not capture on eth0") {
		t.Error("expected warning message text in view")
	}
	// lipgloss renders Foreground(214) + Bold() as the ANSI sequence below.
	// Asserting it pins the visual style so a regression (e.g. dropping the
	// case back to the white default) is caught automatically.
	const amberBoldAnsi = "\x1b[1;38;5;214m"
	if !strings.Contains(view, amberBoldAnsi) {
		t.Errorf("expected warning text to be rendered with amber+bold ANSI %q, got view without it", amberBoldAnsi)
	}
}
