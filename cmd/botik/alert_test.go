package main

import (
	"botik/cmd/botik/alert"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlertOk(t *testing.T) {
	al1 := &alert.Alert{
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

	s, err := getMsg(al1, true)

	assert.NoError(t, err)
	fmt.Println(s)
}

func TestAlertBad(t *testing.T) {
	al1 := &alert.Alert{
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

	al2 := &alert.Alert{
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
		}{},
		ActiveAt: time.Now(),
	}

	s, err := getMsg(al1, false)

	assert.NoError(t, err)
	fmt.Println(s)

	s, err = getMsg(al2, false)

	assert.NoError(t, err)
	fmt.Println(s)
}
