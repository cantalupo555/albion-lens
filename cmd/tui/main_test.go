package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cantalupo555/albion-lens/internal/tui"
	"github.com/cantalupo555/albion-lens/pkg/backend"
	"github.com/cantalupo555/albion-lens/pkg/photon"
)

// ============================================
// bridgeEvents tests
// ============================================

func TestBridgeEventsBatching(t *testing.T) {
	eventsCh := make(chan backend.GameEvent, 10)
	bulkCh := make(chan tui.BulkEventMsg, 5)

	go bridgeEvents(eventsCh, bulkCh, func() *photon.Stats { return nil })
	defer close(eventsCh)

	eventsCh <- backend.GameEvent{Type: backend.EventTypeInfo, Message: "event1"}
	eventsCh <- backend.GameEvent{Type: backend.EventTypeInfo, Message: "event2"}

	select {
	case batch := <-bulkCh:
		if len(batch) != 2 {
			t.Errorf("expected 2 events in batch, got %d", len(batch))
		}
		if batch[0].Message != "event1" {
			t.Errorf("expected first event message 'event1', got %q", batch[0].Message)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for timer-flushed batch")
	}
}

func TestBridgeEventsFlushOnBatchFull(t *testing.T) {
	eventsCh := make(chan backend.GameEvent, 200)
	bulkCh := make(chan tui.BulkEventMsg, 5)

	go bridgeEvents(eventsCh, bulkCh, func() *photon.Stats { return nil })
	defer close(eventsCh)

	for i := 0; i < eventBatchSize; i++ {
		eventsCh <- backend.GameEvent{Type: backend.EventTypeInfo, Message: "batch event"}
	}

	select {
	case batch := <-bulkCh:
		if len(batch) != eventBatchSize {
			t.Errorf("expected %d events in full batch, got %d", eventBatchSize, len(batch))
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for full-batch flush")
	}
}

func TestBridgeEventsChannelClose(t *testing.T) {
	eventsCh := make(chan backend.GameEvent, 10)
	bulkCh := make(chan tui.BulkEventMsg, 5)

	done := make(chan struct{})
	go func() {
		bridgeEvents(eventsCh, bulkCh, func() *photon.Stats { return nil })
		close(done)
	}()

	eventsCh <- backend.GameEvent{Type: backend.EventTypeInfo, Message: "final event"}
	close(eventsCh)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("bridgeEvents did not return after channel close")
	}

	// Final flush should have delivered the buffered event
	select {
	case batch := <-bulkCh:
		if len(batch) != 1 {
			t.Errorf("expected 1 event in final flush, got %d", len(batch))
		}
	case <-time.After(200 * time.Millisecond):
		// Acceptable if already flushed via timer
	}
}

func TestBridgeEventsDropWhenFull(t *testing.T) {
	eventsCh := make(chan backend.GameEvent, 200)
	bulkCh := make(chan tui.BulkEventMsg, 1)

	stats := photon.NewStats()
	go bridgeEvents(eventsCh, bulkCh, func() *photon.Stats { return stats })

	// Fill bulkCh to capacity so sends fail
	bulkCh <- tui.BulkEventMsg{{Type: "blocker", Message: "filler"}}

	// Send enough events to fill the buffer and trigger a flush
	for i := 0; i < eventBatchSize; i++ {
		eventsCh <- backend.GameEvent{Type: backend.EventTypeInfo, Message: "to be dropped"}
	}

	// Wait for drops to be counted (poll up to 1 second)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if stats.GetEventsDropped() >= eventBatchSize {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	dropped := stats.GetEventsDropped()
	if dropped == 0 {
		t.Error("expected dropped events > 0 when bulk channel is full")
	}

	close(eventsCh)
}

func TestBridgeEventsEmptyFlushSendsNothing(t *testing.T) {
	eventsCh := make(chan backend.GameEvent, 10)
	bulkCh := make(chan tui.BulkEventMsg, 5)

	go bridgeEvents(eventsCh, bulkCh, func() *photon.Stats { return nil })

	// Wait for at least one flush interval with no events
	time.Sleep(eventFlushInterval * 2)

	select {
	case <-bulkCh:
		t.Error("expected no batch when buffer is empty")
	default:
		// Success: nothing sent
	}

	close(eventsCh)
}

// ============================================
// Constants tests
// ============================================

func TestConstantsHaveExpectedValues(t *testing.T) {
	if bulkEventChannelSize != 5 {
		t.Errorf("expected bulkEventChannelSize=5, got %d", bulkEventChannelSize)
	}
	if statsChannelSize != 10 {
		t.Errorf("expected statsChannelSize=10, got %d", statsChannelSize)
	}
	if eventBatchSize != 50 {
		t.Errorf("expected eventBatchSize=50, got %d", eventBatchSize)
	}
	if eventFlushInterval != 50*time.Millisecond {
		t.Errorf("expected eventFlushInterval=50ms, got %v", eventFlushInterval)
	}
	if saveInterval != 5*time.Minute {
		t.Errorf("expected saveInterval=5m, got %v", saveInterval)
	}
}

// ============================================
// periodicSave tests
// ============================================

func TestPeriodicSaveFiresAndStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32

	done := make(chan struct{})
	go func() {
		periodicSave(ctx, 10*time.Millisecond, func() error {
			calls.Add(1)
			return nil
		}, nil)
		close(done)
	}()

	// Allow at least one tick to fire.
	time.Sleep(35 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// goroutine exited cleanly after cancel
	case <-time.After(time.Second):
		t.Fatal("periodicSave did not exit after cancel")
	}

	if calls.Load() < 1 {
		t.Errorf("expected periodic save to fire at least once, got %d", calls.Load())
	}
}

func TestPeriodicSaveContinuesAfterError(t *testing.T) {
	// A failing save must not stop the loop; it should keep ticking until
	// cancelled so the user can recover if the filesystem issue is transient.
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32

	done := make(chan struct{})
	go func() {
		periodicSave(ctx, 10*time.Millisecond, func() error {
			calls.Add(1)
			return errBoom
		}, nil)
		close(done)
	}()

	time.Sleep(35 * time.Millisecond)
	cancel()
	<-done

	if calls.Load() < 2 {
		t.Errorf("expected periodic save to keep firing after errors, got %d calls", calls.Load())
	}
}

// errBoom is a sentinel error used only by TestPeriodicSaveContinuesAfterError.
var errBoom = errors.New("boom")
