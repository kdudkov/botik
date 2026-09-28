package answer

import (
	"log/slog"
	"testing"

	"botik/cmd/botik/alert"
)

func TestMuteWithoutAlertReply(t *testing.T) {
	am := New()
	manager := alert.NewManager(slog.Default(), func(string, []string) error { return nil })
	if err := am.RegisterAnswer("alerts", NewAlerts(slog.Default(), manager)); err != nil {
		t.Fatal(err)
	}
	for _, reply := range []string{"", "unrelated reply", "id:"} {
		ans := am.CheckAnswer("user", "mute", reply)
		if ans == nil || ans.Msg == "" {
			t.Fatalf("missing usage response for %q", reply)
		}
	}
}
