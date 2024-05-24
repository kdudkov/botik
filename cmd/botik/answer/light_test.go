package answer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLightCheck(t *testing.T) {
	l := Light{}

	for _, s := range []string{"выключить вeсь свет", "выключи свет везде", "включи свет на кухне"} {
		q := l.Check("", s)
		assert.True(t, q.Matched, "must work on %s", s)
	}
}

func TestPrefix(t *testing.T) {
	s := LongestPrefix("aa bb cc dd", "aa", "aa bb cc", "aa bb")
	assert.Equal(t, "aa bb cc", s, "wrong prefix")
}
