package main

import (
	"botik/cmd/botik/alert"
	"fmt"
	"testing"
	"time"
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

	fmt.Println(getMsg(al1, true))
	fmt.Println(getMsg(al1, false))
	fmt.Println(getMsg(al2, false))
}
