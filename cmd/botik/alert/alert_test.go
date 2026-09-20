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
		Labels:      map[string]string{"alertname": "alert1", "severity": "critical", "host": "host"},
		Annotations: map[string]string{"summary": "summary", "description": "description"},
		StartsAt:    time.Now(),
	}

	al2 := &Alert{
		Labels:      map[string]string{"alertname": "alert2", "severity": "critical", "host": "host"},
		Annotations: map[string]string{"summary": "summary", "description": "description"},
		StartsAt:    time.Now(),
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
