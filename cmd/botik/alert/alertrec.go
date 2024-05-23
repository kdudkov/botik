package alert

import (
	"sync"
	"time"
)

const notifyDelay = time.Hour * 3

type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	GroupID     string            `json:"group_id"`
	Expression  string            `json:"expression"`
	State       string            `json:"state"`
	Value       string            `json:"value"`
	Labels      map[string]string `json:"labels"`
	Annotations struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
	} `json:"annotations"`
	ActiveAt time.Time `json:"activeAt"`
}

type AlertRec struct {
	alert      *Alert
	url        string
	lastNotify time.Time
	muted      bool
	msgId      int
	mx         sync.RWMutex
}

type AlertRecDTO struct {
	Alert      *Alert    `json:"alert,omitempty"`
	Url        string    `json:"url,omitempty"`
	LastNotify time.Time `json:"last_notify"`
	Muted      bool      `json:"muted,omitempty"`
}

func NewAlertRec(alert *Alert, url string) *AlertRec {
	return &AlertRec{
		alert:      alert,
		url:        url,
		lastNotify: time.Time{},
		muted:      false,
		msgId:      0,
		mx:         sync.RWMutex{},
	}
}

func (a *AlertRec) Alert() *Alert {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.alert
}

func (a *AlertRec) SetAlert(alert *Alert) *AlertRec {
	a.mx.Lock()
	defer a.mx.Unlock()

	a.alert = alert

	return a
}

func (a *AlertRec) Notified(m int) *AlertRec {
	a.mx.Lock()
	defer a.mx.Unlock()

	a.msgId = m
	a.lastNotify = time.Now()

	return a
}

func (a *AlertRec) Mute() *AlertRec {
	a.mx.Lock()
	defer a.mx.Unlock()

	a.muted = true

	return a
}

func (a *AlertRec) Url() string {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.url
}

func (a *AlertRec) LastNotify() time.Time {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.lastNotify
}

func (a *AlertRec) DTO() *AlertRecDTO {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return &AlertRecDTO{
		Alert:      a.alert,
		Url:        a.url,
		LastNotify: a.lastNotify,
		Muted:      a.muted,
	}
}

func (a *AlertRec) IsMuted() bool {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.muted
}

func (a *AlertRec) MsgId() int {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.msgId
}

func (a *AlertRec) NeedToNotify() bool {
	a.mx.RLock()
	defer a.mx.RUnlock()

	if a.muted {
		return false
	}

	var severity string

	for k, v := range a.alert.Labels {
		if k == "severity" {
			severity = v
			break
		}
	}

	if severity != "critical" {
		return false
	}

	return time.Now().After(a.lastNotify.Add(notifyDelay))
}
