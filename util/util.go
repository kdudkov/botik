package util

import (
	"fmt"
	"time"
)

func FormatTime(t time.Time) string {
	return fmt.Sprintf("%.2d.%.2d.%.4d %.2d:%.2d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
}

func IsInArray(str string, array []string) bool {
	for _, s1 := range array {
		if str == s1 {
			return true
		}
	}
	return false
}

func HasAllKeys(m map[string]interface{}, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}

	return true
}
