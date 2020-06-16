package answer

import (
	"testing"

	testify_assert "github.com/stretchr/testify/assert"
)

func TestLightCheck(t *testing.T) {
	assert := testify_assert.New(t)

	l := Light{}

	for _, s := range []string{"выключить вeсь свет", "выключи свет везде", "включи свет на кухне"} {
		q := l.Check("", s)
		assert.Truef(q.Matched, "must work on %s", s)
	}
}

func TestPrefix(t *testing.T) {
	assert := testify_assert.New(t)

	s := HasPrefix("aa bb cc dd", "aa", "aa bb cc", "aa bb")
	assert.Equal("aa bb cc", s, "wrong prefix")
}
