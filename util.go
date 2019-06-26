package main

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

const (
	LOG_DEBUG = iota
	LOG_INFO
	LOG_WARN
	LOG_ERROR
)

func FormatTime(t time.Time) string {
	return fmt.Sprintf("%.2d.%.2d.%.4d %.2d:%.2d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
}

func Logf(logger *zap.SugaredLogger, level int8, template string, args ...interface{}) {
	if logger != nil {
		switch level {
		case LOG_DEBUG:
			logger.Debugf(template, args)
		case LOG_INFO:
			logger.Infof(template, args)
		case LOG_WARN:
			logger.Warnf(template, args)
		case LOG_ERROR:
			logger.Errorf(template, args)
		}
	}
}

func Logw(logger *zap.SugaredLogger, level int8, template string, args ...interface{}) {
	if logger != nil {
		switch level {
		case LOG_DEBUG:
			logger.Debugw(template, args)
		case LOG_INFO:
			logger.Infow(template, args)
		case LOG_WARN:
			logger.Warnw(template, args)
		case LOG_ERROR:
			logger.Errorw(template, args)
		}
	}
}

func IsInArray(str string, array []string) bool {
	for _, s1 := range array {
		if str == s1 {
			return true
		}
	}
	return false
}
