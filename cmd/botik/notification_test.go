package main

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type notificationTransport func(*http.Request) (*http.Response, error)

func (f notificationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTelegramDeliveryErrors(t *testing.T) {
	for _, transportFailure := range []bool{false, true} {
		client := &http.Client{Transport: notificationTransport(func(r *http.Request) (*http.Response, error) {
			if transportFailure {
				return nil, errors.New("connection failed")
			}
			return &http.Response{StatusCode: 400, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(
				`{"ok":false,"error_code":400,"description":"Bad Request"}`))}, nil
		})}
		bot := &tg.BotAPI{Token: "test", Client: client}
		bot.SetAPIEndpoint(tg.APIEndpoint)
		app := &App{bot: bot, logger: slog.Default(), conf: NewAppConfig()}
		_ = app.conf.k.Set("notify", []string{"test"})
		_ = app.conf.k.Set("users", map[string]int{"test": 1})
		if _, err := app.sendTgWithMode(1, "alert", "HTML"); err == nil {
			t.Fatal("Telegram error swallowed")
		}
		if err := app.alertNotifier("alert", nil); err == nil {
			t.Fatal("notifier acknowledged failed delivery")
		}
	}
}

type notificationBody struct {
	io.Reader
	closed bool
}

func (b *notificationBody) Close() error { b.closed = true; return nil }

func TestNtfyDeliveryErrors(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	for _, status := range []int{200, 503} {
		body := &notificationBody{Reader: strings.NewReader("")}
		http.DefaultTransport = notificationTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: body, Header: make(http.Header)}, nil
		})
		app := &App{conf: NewAppConfig(), logger: slog.Default()}
		_ = app.conf.k.Set("ntfy.topic", "test")
		err := app.ntfyNotifier("alert", nil)
		if (err != nil) != (status == 503) {
			t.Fatalf("status %d: %v", status, err)
		}
		if !body.closed {
			t.Fatal("response body left open")
		}
	}
}
