package answer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLightCheck(t *testing.T) {
	l := Light{}

	for _, s := range []string{"выключить весь свет", "выключи свет везде", "выключи свет на кухне"} {
		q := l.Check("", s, "")
		assert.True(t, q.Matched, "must work on %s", s)
		assert.Equal(t, OFF, q.Cmd)
	}
}

func TestLightCheckOut(t *testing.T) {
	l := Light{}

	for _, s := range []string{"включи весь свет", "включи свет везде", "включи снаружи", "включи на улице"} {
		q := l.Check("", s, "")
		assert.True(t, q.Matched, "must work on %s", s)
		assert.Equal(t, ON, q.Cmd, "must work on %s", s)
		assert.Equal(t, "lights_out", q.Payload, "must work on %s", s)
	}
}

func TestPrefix(t *testing.T) {
	s := LongestPrefix("aa bb cc dd", "aa", "aa bb cc", "aa bb")
	assert.Equal(t, "aa bb cc", s, "wrong prefix")
}
