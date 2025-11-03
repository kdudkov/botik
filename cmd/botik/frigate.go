package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Review struct {
	Type   string      `json:"type"`
	Before *ReviewInfo `json:"before"`
	After  *ReviewInfo `json:"after"`
}

type ObjectsData struct {
	Detections []string `json:"detections"`
	Objects    []string `json:"objects"`
	SubLabels  []string `json:"sub_labels"`
	Zones      []string `json:"zones"`
	Audio      []any    `json:"audio"`
}

type ReviewInfo struct {
	ID        string       `json:"id"`
	Camera    string       `json:"camera"`
	StartTime float64      `json:"start_time"`
	EndTime   float64      `json:"end_time,omitempty"`
	Severity  string       `json:"severity"`
	ThumbPath string       `json:"thumb_path"`
	Data      *ObjectsData `json:"data"`
}

func (app *App) ProcessReview(b []byte) error {
	review := new(Review)

	if err := json.Unmarshal(b, &review); err != nil {
		return err
	}

	if review.After == nil {
		slog.Error("no after")
		return nil
	}

	var msg string

	if (review.Type == "new" && review.After.Severity == "alert") || (review.Type == "update" && review.Before.Severity != "alert" && review.After.Severity == "alert") {
		msg = fmt.Sprintf("New alert cam: %s, zones: %s, objects: %s", review.After.Camera, strings.Join(review.After.Data.Zones, ","), strings.Join(review.After.Data.Objects, ","))
	}

	if msg == "" {
		return nil
	}

	for _, user := range app.conf.Strings("notify") {
		id, err := app.IdByName(user)

		if err != nil {
			app.logger.Error("invalid user "+user, slog.Any("error", err))
			continue
		}

		msg := tg.NewMessage(id, msg)

		if _, err := app.bot.Send(msg); err != nil {
			app.logger.Error("can't send message", slog.Any("error", err))
		}
		
		if review.After.ThumbPath != "" {
			f,_ := strings.CutPrefix(review.After.ThumbPath, "/media/frigate")
			f = "/home/kott/frigate/storage" + f
			msg2 := tg.NewPhoto(id, tg.FilePath(f))

			if _, err := app.bot.Send(msg2); err != nil {
				app.logger.Error("can't send message", slog.Any("error", err))
			}
		}
	}
	
	return nil
}
