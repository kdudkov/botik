package main

import (
	"fmt"
	"time"
)

func FormatTime(t time.Time) string {
	return fmt.Sprintf("%.2d.%.2d.%.4d %.2d:%.2d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
}
