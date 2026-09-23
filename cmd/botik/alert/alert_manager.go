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
	chIn     chan *Notification
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
		chIn:     make(chan *Notification, 64),
		tpl:      tmpl,
		notifier: notifier,
		mx:       sync.Mutex{},
	}
}

func (a *AlertManager) Start(ctx context.Context) {
	go a.loop()
	go a.reminder(ctx)
}

func (a *AlertManager) AddNotification(alert *Alert, typ NotificationType) {
	select {
	case a.chIn <- &Notification{alert: alert, typ: typ}:
		return
	default:
	}
}

func (a *AlertManager) Process(alert *Alert) {
	if obj, loaded := a.alerts.LoadOrStore(alert.Key(), alert); loaded {
		oldAlert := obj.(*Alert)
		wasActive := oldAlert.IsActive()
		oldAlert.Update(alert)

		a.logger.Debug("alert " + alert.String())

		if wasActive && !alert.IsActive() {
			a.logger.Info(fmt.Sprintf("active -> inactive, alert %s", alert.String()))
			a.AddNotification(oldAlert, NotificationGood)
		}

		if !wasActive && alert.IsActive() {
			a.logger.Info(fmt.Sprintf("inactive -> active, alert %s", alert.String()))
			a.AddNotification(oldAlert, NotificationBad)
		}

		if oldAlert.NeedsNotify() {
			a.logger.Info(fmt.Sprintf("remind, alert %s", alert.String()))
			a.AddNotification(oldAlert, NotificationRemind)
		}
	} else {
		if alert.isActive() {
			a.AddNotification(alert, NotificationBad)
		}
	}
}

func (a *AlertManager) loop() {
	for n := range a.chIn {
		if n.alert.IsMuted() {
			continue
		}

		if !n.alert.NeedsNotify() {
			a.logger.Warn(fmt.Sprintf("alert %s changed his mind", n.alert.String()))
			continue
		}

		var msg string
		var err error

		switch n.typ {
		case NotificationGood:
			msg, err = a.getMsg(n.alert, "alert_good")
		case NotificationBad:
			msg, err = a.getMsg(n.alert, "alert_bad")
		case NotificationRemind:
			msg, err = a.getMsg(n.alert, "reminder")
		}

		if err != nil {
			a.logger.Error("template error", "error", err.Error())
			continue
		}

		a.notifier(msg)
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
		if alert.NeedsNotify() {
			if alert.IsActive() {
				a.AddNotification(alert, NotificationRemind)
			} else {
				a.AddNotification(alert, NotificationGood)
			}
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
