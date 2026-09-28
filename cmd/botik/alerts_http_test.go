package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"botik/cmd/botik/alert"
)

func TestAlertsHandlerRejectsOverflow(t *testing.T) {
	manager := alert.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)), func(string, []string) error { return nil })
	app := &App{am: manager}
	server := fiber.New()
	server.Post("/api/v2/alerts", AlertsHandlerFunc(app))
	post := func(body string, want int) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/v2/alerts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res, err := server.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := res.Body.Close(); err != nil {
				t.Error(err)
			}
		}()
		if res.StatusCode != want {
			t.Fatalf("want %d, got %d", want, res.StatusCode)
		}
	}
	post(`[null]`, http.StatusBadRequest)
	post(`[{"labels":{"alertname":"test"}}]`, http.StatusOK)
	for manager.Add(&alert.Alert{}) {
	}
	post(`[{"labels":{"alertname":"test"}}]`, http.StatusServiceUnavailable)
}
