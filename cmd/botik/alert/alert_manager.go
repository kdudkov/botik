package alert

import (
	"context"
	"embed"
	"html/template"
	"log/slog"
	"strings"
	"sync"
	"time"
)

//go:embed template/*
var alerts embed.FS

type AlertManager struct {
	logger   *slog.Logger
	alerts   sync.Map
	chIn     chan *Alert
	tpl      *template.Template
	notifier func(msg string, tags []string) error
}

func NewManager(logger *slog.Logger, notifier func(string, []string) error) *AlertManager {
	tmpl, err := template.New("").ParseFS(alerts, "template/*")

	if err != nil {
		panic(err)
	}

	return &AlertManager{
		logger:   logger,
		alerts:   sync.Map{},
		chIn:     make(chan *Alert, 64),
		tpl:      tmpl,
		notifier: notifier,
	}
}

func (a *AlertManager) Start(ctx context.Context) {
	go a.loop(ctx)
}

func (a *AlertManager) Add(alert *Alert) bool {
	if alert == nil {
		return false
	}
	select {
	case a.chIn <- alert:
		return true
	default:
		return false
	}
}

func (a *AlertManager) process(alert *Alert) {
	if alert == nil {
		return
	}

	var st *AlertState

	key := alert.Key()

	if obj, loaded := a.alerts.Load(key); loaded {
		st = obj.(*AlertState)
	} else {
		initial := &AlertState{id: key}
		obj, _ := a.alerts.LoadOrStore(key, initial)
		st = obj.(*AlertState)
	}

	changed := st.Update(alert)

	var tpl string
	var tags []string
	
	switch {
	case !st.isActive() && changed:
		tpl = "alert_good"
		tags = []string{"green_square", "alerts"}
	case st.isActive() && changed:
		tpl = "alert_bad"
		tags = []string{"warning", "alerts"}
		if alert.Labels["severity"] == "critical" {
			tags = append(tags, "red_square")
		} else {
			tags = append(tags, "yellow_square")
		}
	case !changed && st.needsNotify():
		tpl = "reminder"
		tags = []string{"alarm_clock", "alerts"}
	}

	if tpl != "" {
		msg, err := a.getMsg(alert, tpl)

		if err != nil {
			a.logger.Error("template error", "error", err)
			return
		}

		if err := a.notifier(msg, tags); err != nil {
			a.logger.Error("notification failed", "error", err)
			return
		}

		st.acknowledge()
	}
}

func (a *AlertManager) loop(ctx context.Context) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case alert, ok := <-a.chIn:
			if !ok {
				return
			}
			a.process(alert)
		case <-ctx.Done():
			return
		}
	}
}

func (a *AlertManager) Range(f func(*AlertState) bool) {
	a.alerts.Range(func(_, value any) bool {
		return f(value.(*AlertState))
	})
}

func (a *AlertManager) getMsg(alert *Alert, tpl_name string) (string, error) {
	sb := new(strings.Builder)

	if err := a.tpl.ExecuteTemplate(sb, tpl_name, map[string]any{"alert": map[string]any{
		"Name":        alert.Labels["alertname"],
		"Severity":    alert.Labels["severity"],
		"Description": alert.Annotations["description"],
		"Labels":      alert.Labels,
		"Key":         alert.Key(),
	}}); err != nil {
		return "", err
	}

	return sb.String(), nil
}
