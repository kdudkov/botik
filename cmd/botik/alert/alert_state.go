package alert

import (
	"maps"
	"sync"
	"time"
)

const NotificationTime = time.Minute * 60

type AlertState struct {
	id               string
	alert            *Alert
	active           bool
	muted            bool
	updated          time.Time
	lastNotify       time.Time
	revision         uint64
	mx               sync.RWMutex
}

type AlertStateDTO struct {
	ID           string            `json:"id"`
	Revision     uint64            `json:"revision"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Muted        bool              `json:"muted"`
	Active       bool              `json:"active"`
}

func (a *AlertState) Mute() {
	a.mx.Lock()
	defer a.mx.Unlock()

	a.muted = true
}

func (a *AlertState) isMuted() bool {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.muted
}

func (a *AlertState) isActive() bool {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.active
}

func (a *AlertState) needsNotify() bool {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.active && !a.muted && time.Since(a.lastNotify) > NotificationTime
}

func (a *AlertState) acknowledge() {
	a.mx.Lock()
	defer a.mx.Unlock()

	a.lastNotify = time.Now()
}

func (a *AlertState) Update(alert *Alert) bool {
	if a == nil || alert == nil {
		return false
	}

	a.mx.Lock()
	defer a.mx.Unlock()

	active := alert.isActive()
	changed := a.alert == nil || a.active != active || !a.alert.StartsAt.Equal(alert.StartsAt)
	if changed {
		a.revision++
	}

	a.active = active
	a.alert = alert
	a.updated = time.Now()

	return changed
}

func (a *AlertState) DTO() *AlertStateDTO {
	if a == nil {
		return nil
	}

	a.mx.RLock()
	defer a.mx.RUnlock()

	if a.alert == nil {
		return &AlertStateDTO{ID: a.id, Muted: a.muted}
	}

	return &AlertStateDTO{
		ID:           a.id,
		Revision:     a.revision,
		Labels:       maps.Clone(a.alert.Labels),
		Annotations:  maps.Clone(a.alert.Annotations),
		StartsAt:     a.alert.StartsAt,
		EndsAt:       a.alert.EndsAt,
		GeneratorURL: a.alert.GeneratorURL,
		Muted:        a.muted,
		Active:       a.active,
	}
}
