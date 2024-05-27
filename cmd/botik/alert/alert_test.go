package alert

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlertBad(t *testing.T) {
	am := NewManager(slog.Default(), func(msg string) {

	})

	al1 := &Alert{
		ID:         "id",
		Name:       "alert name",
		GroupID:    "grp",
		Expression: "a == b",
		State:      "Error",
		Value:      "43",
		Labels:     map[string]string{"severity": "critical", "host": "host"},
		Annotations: struct {
			Summary     string `json:"summary"`
			Description string `json:"description"`
		}{Summary: "summary", Description: "description"},
		ActiveAt: time.Now(),
	}

	al2 := &Alert{
		ID:         "id",
		Name:       "alert name",
		GroupID:    "grp",
		Expression: "a == b",
		State:      "Error",
		Value:      "43",
		Labels:     map[string]string{"severity": "bad", "host": "host"},
		Annotations: struct {
			Summary     string `json:"summary"`
			Description string `json:"description"`
		}{},
		ActiveAt: time.Now(),
	}

	for _, tpl := range []string{"alert_bad", "alert_good", "inactive", "reminder"} {
		t.Run("alert_"+tpl, func(t *testing.T) {
			s, err := am.getMsg(al1, tpl)

			assert.NoError(t, err)
			fmt.Println("========= " + tpl)
			fmt.Println(s)

			s, err = am.getMsg(al2, tpl)

			assert.NoError(t, err)
			fmt.Println("========= " + tpl)
			fmt.Println(s)
		})
	}

}
