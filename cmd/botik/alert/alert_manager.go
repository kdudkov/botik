package alert

import (
	"context"
	"embed"
	"fmt"
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
	notifier func(msg string)
	mx       sync.Mutex
}

func NewManager(logger *slog.Logger, notifier func(msg string)) *AlertManager {
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
		mx:       sync.Mutex{},
	}
}

func (a *AlertManager) Start(ctx context.Context) {
	go a.loop()
	go a.reminder(ctx)
}

func (a *AlertManager) Add(alert *Alert) bool {
	select {
	case a.chIn <- alert:
		return true
	default:
		return false
	}
}

func (a *AlertManager) loop() {
	for alert := range a.chIn {
		if obj, loaded := a.alerts.LoadOrStore(alert.Key(), alert); loaded {
			oldAlert := obj.(*Alert)

			oldAlert.Update(alert)

			if oldAlert.NeedsNotify() {
				a.notify(alert, "reminder")
			}
		} else {
			a.notify(alert, "alert_bad")
		}
	}
}

func (a *AlertManager) reminder(ctx context.Context) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			a.remind()
		case <-ctx.Done():
			return
		}
	}
}

func (a *AlertManager) remind() {
	a.Range(func(alert *Alert) bool {
		if !alert.IsActive() {
			a.logger.Info(fmt.Sprintf("alert %s is good", alert.Name()))
			a.notify(alert, "alert_good")

			a.alerts.Delete(alert.Key())
		}

		return true
	})
}

func (a *AlertManager) Range(f func(a *Alert) bool) {
	a.alerts.Range(func(_, value any) bool {
		if alertRec, ok := value.(*Alert); ok {
			return f(alertRec)
		}

		return true
	})
}

func (a *AlertManager) notify(alert *Alert, tpl string) {
	if alert.IsMuted() {
		return
	}

	if msg, err := a.getMsg(alert, tpl); err == nil {
		alert.Notified()
		a.notifier(msg)
	} else {
		a.logger.Error("error in template", "error", err)
	}
}

func (a *AlertManager) getMsg(alert *Alert, tpl_name string) (string, error) {
	sb := new(strings.Builder)

	if err := a.tpl.ExecuteTemplate(sb, tpl_name, map[string]any{"alert": alert}); err != nil {
		a.logger.Error("error in template", "error", err)
		return "", err
	}

	return sb.String(), nil
}
