package alert

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type Alert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

func (alert *Alert) String() string {

	return fmt.Sprintf("Labels: %s, Annotations: %s, StartsAt: %s, EndsAt: %s, GeneratorURL: %s",
		alert.Labels, alert.Annotations, alert.StartsAt, alert.EndsAt, alert.GeneratorURL)
}

func (alert *Alert) Key() string {
	b, _ := json.Marshal(alert.Labels)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (alert *Alert) Name() string {
	return alert.Labels["alertname"]
}

func (alert *Alert) Description() string {
	return alert.Annotations["description"]
}

func (alert *Alert) isActive() bool {
	return alert.EndsAt.IsZero() || alert.EndsAt.After(time.Now())
}
