package main

import (
	"fmt"
	"testing"
	"time"
)

func TestAlertOk(t *testing.T) {
	alert := &Alert{
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

	alert2 := &Alert{
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

	fmt.Println(getMsg(alert, true))
	fmt.Println(getMsg(alert, false))
	fmt.Println(getMsg(alert2, false))
}
