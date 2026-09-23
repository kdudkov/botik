package alert

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"sync"
	"time"
)

const NotificationTime = time.Minute * 60

type Alert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	muted        bool
	lastNotify   time.Time
	mx           sync.RWMutex
}

type AlertDTO struct {
	ID           string            `json:"id"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Muted        bool              `json:"muted"`
	Active       bool              `json:"active"`
}

func (alert *Alert) String() string {
	alert.mx.RLock()
	defer alert.mx.RUnlock()

	return fmt.Sprintf("Labels: %s, Annotations: %s, StartsAt: %s, EndsAt: %s, GeneratorURL: %s",
		alert.Labels, alert.Annotations, alert.StartsAt, alert.EndsAt, alert.GeneratorURL)
}

func (alert *Alert) Key() string {
	alert.mx.RLock()
	defer alert.mx.RUnlock()

	return alert.key()
}

func (alert *Alert) key() string {
	b, _ := json.Marshal(alert.Labels)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (alert *Alert) Equal(alert2 *Alert) bool {
	return maps.Equal(alert.Labels, alert2.Labels)
}

func (alert *Alert) Name() string {
	return alert.Labels["alertname"]
}

func (alert *Alert) Severity() string {
	return alert.Labels["severity"]
}

func (alert *Alert) Instance() string {
	return alert.Labels["instance"]
}

func (alert *Alert) Job() string {
	return alert.Labels["job"]
}

func (alert *Alert) Summary() string {
	alert.mx.RLock()
	defer alert.mx.RUnlock()

	return alert.Annotations["summary"]
}

func (alert *Alert) Description() string {
	alert.mx.RLock()
	defer alert.mx.RUnlock()

	return alert.Annotations["description"]
}

func (alert *Alert) IsActive() bool {
	alert.mx.RLock()
	defer alert.mx.RUnlock()

	return alert.isActive()
}

func (alert *Alert) isActive() bool {
	return alert.EndsAt.After(time.Now())
}

func (alert *Alert) Mute() {
	alert.mx.Lock()
	defer alert.mx.Unlock()

	alert.muted = true
}

func (alert *Alert) IsMuted() bool {
	alert.mx.RLock()
	defer alert.mx.RUnlock()

	return alert.muted
}

func (alert *Alert) Notified() {
	alert.mx.Lock()
	defer alert.mx.Unlock()

	alert.lastNotify = time.Now()
}

func (alert *Alert) NeedsNotify() bool {
	alert.mx.RLock()
	defer alert.mx.RUnlock()

	if alert.muted {
		return false
	}

	// stale but not notified
	if !alert.isActive() && alert.lastNotify.Before(alert.EndsAt) {
		return true
	}

	return alert.isActive() && time.Since(alert.lastNotify) > NotificationTime
}

func (alert *Alert) Update(a1 *Alert) {
	if alert == nil || a1 == nil {
		return
	}

	alert.mx.RLock()
	defer alert.mx.RUnlock()

	alert.StartsAt = a1.StartsAt
	alert.EndsAt = a1.EndsAt

	alert.Annotations = a1.Annotations
}

func (alert *Alert) DTO() *AlertDTO {
	if alert == nil {
		return nil
	}

	alert.mx.RLock()
	defer alert.mx.RUnlock()

	return &AlertDTO{
		ID:           alert.key(),
		Labels:       alert.Labels,
		Annotations:  alert.Annotations,
		StartsAt:     alert.StartsAt,
		EndsAt:       alert.EndsAt,
		GeneratorURL: alert.GeneratorURL,
		Muted:        alert.muted,
		Active:       alert.isActive(),
	}
}
