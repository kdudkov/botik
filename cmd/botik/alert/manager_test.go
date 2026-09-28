package alert

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func testManager(notifier func(string, []string) error) *AlertManager {
	return NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)), notifier)
}

func testAlert(active bool) *Alert {
	end := time.Now().Add(-time.Minute)
	if active {
		end = time.Now().Add(time.Hour)
	}
	return &Alert{
		Labels:      map[string]string{"alertname": "test"},
		Annotations: map[string]string{"description": "test description"},
		StartsAt:    time.Unix(1000, 0), EndsAt: end,
	}
}

func drain(m *AlertManager) {
	for {
		select {
		case a := <-m.chIn:
			m.process(a)
		default:
			return
		}
	}
}

func TestNotificationTransitions(t *testing.T) {
	var messages []string
	m := testManager(func(msg string, tags []string) error { messages = append(messages, msg); return nil })
	for _, active := range []bool{true, false, true} {
		m.Add(testAlert(active))
	}
	drain(m)
	if len(messages) != 3 {
		t.Fatalf("want firing, resolved, firing; got %v", messages)
	}
	if !strings.Contains(messages[1], "is good") || strings.Contains(messages[2], "is good") || strings.Contains(messages[2], "reminder") {
		t.Fatalf("incorrect transition messages: %v", messages)
	}
	m.Add(testAlert(true))
	drain(m)
	if len(messages) != 3 {
		t.Fatal("unchanged state generated a duplicate")
	}
}

func TestNewStartsAtNotifies(t *testing.T) {
	calls := 0
	m := testManager(func(string, []string) error { calls++; return nil })
	m.process(testAlert(true))
	a := testAlert(true)
	a.StartsAt = a.StartsAt.Add(time.Minute)
	m.process(a)
	if calls != 2 {
		t.Fatalf("new firing lost: %d", calls)
	}
}

func TestUpdateDuringDeliveryIsQueued(t *testing.T) {
	var m *AlertManager
	var messages []string
	m = testManager(func(s string, tags []string) error {
		messages = append(messages, s)
		if len(messages) == 1 {
			m.Add(testAlert(false))
		}
		return nil
	})
	m.Add(testAlert(true))
	drain(m)
	if len(messages) != 2 || !strings.Contains(messages[1], "is good") {
		t.Fatalf("recovery lost: %v", messages)
	}
}

func TestConcurrentAddAndStateReads(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := testManager(func(string, []string) error { return nil })
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		m.Start(ctx)
		var wg sync.WaitGroup
		for worker := 0; worker < 4; worker++ {
			wg.Go(func() {
				for i := 0; i < 100; i++ {
					m.Add(testAlert(i%2 == 0))
					m.Range(func(st *AlertState) bool { _ = st.DTO(); st.Mute(); return true })
				}
			})
		}
		wg.Wait()
		synctest.Wait()
	})
}

func TestLoopStopsOnCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		m := testManager(func(string, []string) error { calls.Add(1); return nil })
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan struct{})
		go func() { m.loop(ctx); close(done) }()
		m.Add(testAlert(true))
		synctest.Wait()
		if calls.Load() != 1 {
			t.Fatalf("initial calls: %d", calls.Load())
		}
		cancel()
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("loop did not stop after cancellation")
		}
	})
}

func TestResolvedAlertHasNoReminder(t *testing.T) {
	calls := 0
	m := testManager(func(string, []string) error { calls++; return nil })
	m.process(testAlert(false))
	m.Range(func(st *AlertState) bool {
		st.lastNotify = time.Now().Add(-2 * NotificationTime)
		return true
	})
	m.process(testAlert(false))
	if calls != 1 {
		t.Fatalf("resolved alert generated a reminder: %d", calls)
	}
}

func TestQueueOverflowIsReported(t *testing.T) {
	m := testManager(func(string, []string) error { return nil })
	for i := 0; i < cap(m.chIn); i++ {
		if !m.Add(testAlert(true)) {
			t.Fatalf("queue full after %d items", i)
		}
	}
	if m.Add(testAlert(false)) {
		t.Fatal("overflow silently accepted")
	}
	if m.Add(nil) {
		t.Fatal("nil alert accepted")
	}
	drain(m)
	if !m.Add(testAlert(false)) {
		t.Fatal("queue did not recover after draining")
	}
}

func TestReminderOnRepeatedActiveAlert(t *testing.T) {
	for _, muted := range []bool{false, true} {
		t.Run(map[bool]string{false: "unmuted", true: "muted"}[muted], func(t *testing.T) {
			var messages []string
			m := testManager(func(s string, tags []string) error { messages = append(messages, s); return nil })
			m.process(testAlert(true))
			m.Range(func(st *AlertState) bool {
				st.lastNotify = time.Now().Add(-2 * NotificationTime)
				if muted {
					st.Mute()
				}
				return true
			})
			m.process(testAlert(true))
			m.process(testAlert(true))
			if muted {
				if len(messages) != 1 {
					t.Fatalf("muted reminder: %v", messages)
				}
			} else if len(messages) != 2 || !strings.Contains(messages[1], "reminder") {
				t.Fatalf("want exactly one reminder: %v", messages)
			}
		})
	}
}
