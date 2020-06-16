package answer

import (
	testify_assert "github.com/stretchr/testify/assert"
	"testing"
)

func TestWords(t *testing.T) {
	assert := testify_assert.New(t)

	var q = Q{}
	var w []string

	q.Msg = " message,  number one!   5,25\n"
	w = q.Words()

	assert.Equal("message", w[0], "error")
	assert.Equal("number", w[1], "error")
	assert.Equal("one", w[2], "error")
	assert.Equal("5,25", w[3], "error")
}
