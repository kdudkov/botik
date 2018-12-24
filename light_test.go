package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLightCheck(t *testing.T) {
	assert := assert.New(t)

	l := Light{}

	for _, s := range []string{"выключить всь свет", "выключи свет везде", "включи свет на кухне"} {
		q := l.Check(s)
		assert.Truef(q.Matched, "must work on %s", s)
	}
}

func TestPrefix(t *testing.T) {
	assert := assert.New(t)

	s := hasPrefix("aa bb cc dd", "aa", "aa bb cc", "aa bb")
	assert.Equal("aa bb cc", s, "wrong prefix")
}
